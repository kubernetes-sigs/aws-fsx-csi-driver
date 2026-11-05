#!/usr/bin/env python3

# Copyright 2026 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Bump the CSI sidecar images to their latest eksbuild tags.

Each sidecar is pinned as public.ecr.aws/csi-components/<name>:vX.Y.Z-eksbuild.N
in the Helm values and in the kustomize base manifests. This script asks the
public ECR registry which tags exist for each pinned sidecar and moves it to the
highest eksbuild revision of the upstream version it is already on.

Upstream version changes (the vX.Y.Z part) are reported but never applied: those
tend to come with flag or behaviour changes that need a human to review.
"""

import argparse
import json
import os
import re
import sys
import time
import urllib.error
import urllib.request

NAMESPACE = "csi-components"

FILES = [
    "charts/aws-fsx-csi-driver/values.yaml",
    "deploy/kubernetes/base/controller-deployment.yaml",
    "deploy/kubernetes/base/node-daemonset.yaml",
]

# v2.19.0-eksbuild.7 -> (2, 19, 0, 7)
TAG_RE = re.compile(r"^v(\d+)\.(\d+)\.(\d+)-eksbuild\.(\d+)$")

# deploy manifests: image: public.ecr.aws/csi-components/csi-provisioner:v6.3.0-eksbuild.7
INLINE_RE = re.compile(r"(public\.ecr\.aws/" + NAMESPACE + r"/([\w.-]+)):(\S+)")

# chart values: repository: public.ecr.aws/csi-components/csi-provisioner
#                 tag: v6.3.0-eksbuild.7
REPO_RE = re.compile(r"repository:\s*public\.ecr\.aws/" + NAMESPACE + r"/([\w.-]+)\s*$")
VALUES_TAG_RE = re.compile(r"^(\s*tag:\s*)(\S+)(.*)$")


def parse_tag(tag):
    m = TAG_RE.match(tag)
    return tuple(int(g) for g in m.groups()) if m else None


def fetch(url, headers=None, attempts=5):
    """GET a URL, retrying on the registry's rate limiting."""
    for attempt in range(attempts):
        try:
            req = urllib.request.Request(url, headers=headers or {})
            with urllib.request.urlopen(req, timeout=30) as resp:
                body = json.loads(resp.read())
            if "errors" not in body:
                return body
            codes = {e.get("code") for e in body["errors"]}
            if "TOOMANYREQUESTS" not in codes:
                raise RuntimeError("{}: {}".format(url, body["errors"]))
        except urllib.error.HTTPError as err:
            if err.code not in (429, 503):
                raise
        time.sleep(2 * (attempt + 1))
    raise RuntimeError("giving up on {} after {} attempts".format(url, attempts))


def list_tags(name):
    repo = "{}/{}".format(NAMESPACE, name)
    token = fetch(
        "https://public.ecr.aws/token/?scope=repository:{}:pull"
        "&service=public.ecr.aws".format(repo)
    )["token"]
    body = fetch(
        "https://public.ecr.aws/v2/{}/tags/list".format(repo),
        {"Authorization": "Bearer " + token},
    )
    return body.get("tags") or []


def discover_pins(files):
    """Map sidecar name -> set of tags it is currently pinned to."""
    pins = {}
    for path in files:
        pending = None
        for line in open(path):
            for _, name, tag in INLINE_RE.findall(line):
                pins.setdefault(name, set()).add(tag.rstrip('"\''))
            repo = REPO_RE.search(line)
            if repo:
                pending = repo.group(1)
                continue
            if pending:
                tag = VALUES_TAG_RE.match(line)
                if tag:
                    pins.setdefault(pending, set()).add(tag.group(2).strip('"\''))
                    pending = None
    return pins


def resolve(name, current_tags):
    """Pick the newest eksbuild tag on the same upstream version as `current`."""
    parsed = {t: parse_tag(t) for t in current_tags}
    unparsed = sorted(t for t, p in parsed.items() if p is None)
    if unparsed:
        raise RuntimeError(
            "{}: cannot parse pinned tag(s) {}".format(name, ", ".join(unparsed))
        )
    # A sidecar should be pinned to one tag everywhere; if not, start from the oldest.
    current = min(current_tags, key=lambda t: parsed[t])
    cur = parsed[current]

    available = [(t, parse_tag(t)) for t in list_tags(name)]
    available = [(t, p) for t, p in available if p is not None]

    same_upstream = [(t, p) for t, p in available if p[:3] == cur[:3]]
    newest = max(same_upstream, key=lambda tp: tp[1])[0] if same_upstream else current

    newer_upstream = sorted({t for t, p in available if p[:3] > cur[:3]})
    return current, newest, newer_upstream


def rewrite(path, moves, dry_run):
    """Apply name -> new tag moves to a file. Returns number of lines changed."""
    out, changed, pending = [], 0, None
    for line in open(path):
        original = line

        def swap_inline(m):
            new = moves.get(m.group(2))
            return "{}:{}".format(m.group(1), new) if new else m.group(0)

        line = INLINE_RE.sub(swap_inline, line)

        repo = REPO_RE.search(line)
        if repo:
            pending = repo.group(1)
        elif pending:
            tag = VALUES_TAG_RE.match(line)
            if tag:
                new = moves.get(pending)
                if new:
                    line = "{}{}{}\n".format(tag.group(1), new, tag.group(3).rstrip())
                pending = None

        if line != original:
            changed += 1
        out.append(line)

    if changed and not dry_run:
        with open(path, "w") as fh:
            fh.writelines(out)
    return changed


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="report what would change without writing any files",
    )
    parser.add_argument(
        "--summary-file",
        help="write a markdown summary of the bumps here (for PR bodies)",
    )
    args = parser.parse_args()

    missing = [f for f in FILES if not os.path.exists(f)]
    if missing:
        sys.exit("run me from the repository root; missing {}".format(missing[0]))

    pins = discover_pins(FILES)
    if not pins:
        sys.exit("found no public.ecr.aws/{} pins to bump".format(NAMESPACE))

    moves, held_back, rows = {}, [], []
    for name in sorted(pins):
        current, newest, newer_upstream = resolve(name, pins[name])
        if newest != current:
            moves[name] = newest
            rows.append((name, current, newest))
        if newer_upstream:
            held_back.append((name, current, newer_upstream[-1]))

    if not moves:
        print("all sidecars are already on their latest eksbuild tag")
    for name, current, newest in rows:
        print("{:26s} {} -> {}".format(name, current, newest))

    touched = 0
    if moves:
        for path in FILES:
            n = rewrite(path, moves, args.dry_run)
            if n:
                touched += 1
                print("  {} ({} line{})".format(path, n, "" if n == 1 else "s"))

    for name, current, latest in held_back:
        print(
            "note: {} has a newer upstream release {} (pinned {}); "
            "left alone, bump it by hand".format(name, latest, current),
            file=sys.stderr,
        )

    if args.summary_file:
        lines = []
        if rows:
            lines.append("Bump the CSI sidecar images to the latest eksbuild tags:")
            lines.append("")
            width = max(len(n) for n, _, _ in rows)
            for name, current, newest in rows:
                lines.append(
                    "    {:{w}s} {} -> {}".format(name, current, newest, w=width)
                )
        else:
            lines.append("All sidecars were already on their latest eksbuild tags.")
        if held_back:
            lines.append("")
            lines.append("Newer upstream releases exist and were **not** applied:")
            lines.append("")
            for name, current, latest in held_back:
                lines.append("  * `{}`: pinned `{}`, latest `{}`".format(name, current, latest))
        with open(args.summary_file, "w") as fh:
            fh.write("\n".join(lines) + "\n")

    # Signal to CI whether anything changed.
    if os.environ.get("GITHUB_OUTPUT"):
        with open(os.environ["GITHUB_OUTPUT"], "a") as fh:
            fh.write("changed={}\n".format("true" if moves else "false"))

    print("\n{} file(s) updated{}".format(touched, " (dry run)" if args.dry_run else ""))


if __name__ == "__main__":
    main()
