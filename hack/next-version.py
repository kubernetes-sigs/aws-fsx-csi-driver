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

"""Work out the version of the next release from what landed since the last tag.

  * nothing but CSI sidecar image tag bumps  ->  patch release (v1.11.0 -> v1.11.1)
  * anything else                            ->  minor release (v1.11.0 -> v1.12.0)

Release plumbing commits ("Release 1.11.0 part 1/3" and friends) are ignored
either way. The rule is deliberately narrow: a commit only counts as a sidecar
bump when every line it changes is a sidecar image tag in one of the three files
those tags live in. Anything else - including a Go dependency refresh - makes it
a minor release, because guessing high is the cheaper mistake.

Both kinds of release get a branch cut from master, named after the version:
release-1.12 for a minor, release-1.11.1 for a patch. A patch cannot reuse the
existing release-1.11 branch, because the commits being released are on master
and release-1.11 has already diverged from it.
"""

import argparse
import json
import os
import re
import subprocess
import sys

VERSION_RE = re.compile(r"^v(\d+)\.(\d+)\.(\d+)$")

SKIP_SUBJECT_RE = re.compile(r"^(Release\b|\d+\.\d+ Release\b)|\b[123]/3\b", re.I)

SIDECAR_FILES = {
    "charts/aws-fsx-csi-driver/values.yaml",
    "deploy/kubernetes/base/controller-deployment.yaml",
    "deploy/kubernetes/base/node-daemonset.yaml",
}

# A changed line that is nothing but a sidecar image pin.
SIDECAR_LINE_RE = re.compile(
    r"^[-+]\s*(-\s*)?(image:\s*public\.ecr\.aws/csi-components/[\w.-]+:\S+"
    r"|tag:\s*\S+)\s*$"
)


def run(*args):
    done = subprocess.run(args, capture_output=True, text=True)
    if done.returncode:
        sys.exit("{} failed: {}".format(" ".join(args), done.stderr.strip()))
    return done.stdout


def latest_tag():
    tags = [t for t in run("git", "tag", "--sort=-v:refname").split("\n") if VERSION_RE.match(t)]
    if not tags:
        sys.exit("no vX.Y.Z tags found, cannot work out the next version")
    return tags[0]


def commit_files(sha):
    out = run("git", "show", "--pretty=format:", "--name-only", sha)
    return {f for f in out.split("\n") if f.strip()}


def is_sidecar_only(sha):
    files = commit_files(sha)
    if not files or not files <= SIDECAR_FILES:
        return False
    diff = run("git", "show", "--pretty=format:", "--unified=0", sha)
    for line in diff.split("\n"):
        if not line or line[0] not in "+-" or line[:3] in ("+++", "---"):
            continue
        if not SIDECAR_LINE_RE.match(line):
            return False
    return True


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--previous-version", help="defaults to the newest vX.Y.Z tag")
    parser.add_argument("--ref", default="HEAD", help="what to compare against (default HEAD)")
    parser.add_argument("--json", action="store_true", help="print the result as JSON")
    args = parser.parse_args()

    previous = args.previous_version or latest_tag()
    if not VERSION_RE.match(previous):
        sys.exit("--previous-version must look like v1.11.0, got {}".format(previous))

    log = run(
        "git", "log", "{}..{}".format(previous, args.ref), "--no-merges", "--pretty=%H\x1f%s"
    )

    plumbing, sidecar, real = [], [], []
    for line in log.split("\n"):
        if not line.strip():
            continue
        sha, subject = line.split("\x1f")
        if SKIP_SUBJECT_RE.search(subject):
            plumbing.append((sha, subject))
        elif is_sidecar_only(sha):
            sidecar.append((sha, subject))
        else:
            real.append((sha, subject))

    major, minor, patch = (int(g) for g in VERSION_RE.match(previous).groups())
    if real:
        bump, version = "minor", "v{}.{}.0".format(major, minor + 1)
        branch = "release-{}.{}".format(major, minor + 1)
    else:
        bump, version = "patch", "v{}.{}.{}".format(major, minor, patch + 1)
        branch = "release-{}.{}.{}".format(major, minor, patch + 1)
        if not sidecar:
            print(
                "note: nothing but release plumbing landed since {}, so the release "
                "is whatever this run bumps".format(previous),
                file=sys.stderr,
            )

    result = {
        "previous_version": previous,
        "version": version,
        "bump": bump,
        "branch": branch,
        "real_commits": len(real),
        "sidecar_commits": len(sidecar),
        "plumbing_commits": len(plumbing),
    }

    if args.json:
        print(json.dumps(result, indent=2))
    else:
        print("previous release : {}".format(previous))
        print("next release     : {}  ({} bump)".format(version, bump))
        print("release branch   : {}  (cut from master)".format(branch))
        print()
        for label, commits in (
            ("changes", real),
            ("sidecar bumps", sidecar),
            ("release plumbing (ignored)", plumbing),
        ):
            if commits:
                print("{} ({}):".format(label, len(commits)))
                for sha, subject in commits:
                    print("  {} {}".format(sha[:8], subject))

    if os.environ.get("GITHUB_OUTPUT"):
        with open(os.environ["GITHUB_OUTPUT"], "a") as fh:
            for key, value in result.items():
                fh.write("{}={}\n".format(key, str(value).lower() if isinstance(value, bool) else value))


if __name__ == "__main__":
    main()
