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

"""Apply the "Release X.Y.Z part 1/3" edits.

Part 1/3 of a release touches four files:

  * Makefile          - the VERSION the driver reports and the image is tagged with
  * CHANGELOG-0.x.md  - a new section listing what landed since the previous tag
  * docs/README.md    - a new row in each of the three compatibility matrices
  * docs/install.md   - the kustomize ?ref= pointer for the new release branch

Changelog authors come from the GitHub commits API, which returns the GitHub
login. The git author name is only a fallback: it is often a person's real name,
which does not make a valid https://github.com/<user> link.

Every edit is idempotent, so re-running on a prepared tree is a no-op.
"""

import argparse
import json
import os
import re
import subprocess
import sys
import urllib.error
import urllib.request

CHANGELOG = "CHANGELOG-0.x.md"
MAKEFILE = "Makefile"
README = "docs/README.md"
INSTALL = "docs/install.md"

VERSION_RE = re.compile(r"^v(\d+)\.(\d+)\.(\d+)$")
SEPARATOR_RE = re.compile(r"^\|[-|]+\|\s*$")
LOGIN_RE = re.compile(r"^[A-Za-z0-9][A-Za-z0-9-]*$")

# Release plumbing commits are not release notes.
SKIP_SUBJECT_RE = re.compile(r"^(Release\b|\d+\.\d+ Release\b)|\b[123]/3\b", re.I)

IMAGE_REPO = "public.ecr.aws/fsx-csi-driver/aws-fsx-csi-driver"

# Each matrix gets a new top row. "copy" reuses the previous row's support
# columns; the version cell is always replaced.
MATRICES = [
    ("### CSI Specification Compatibility Matrix", "copy"),
    ("### Kubernetes Version Compatibility Matrix", "copy"),
    ("### Container Images", "image"),
]


def run(*args):
    return subprocess.run(args, capture_output=True, text=True, check=True).stdout


def git_tags():
    out = run("git", "tag", "--sort=-v:refname")
    return [t for t in out.split("\n") if VERSION_RE.match(t)]


def previous_version(version):
    for tag in git_tags():
        if tag != version:
            return tag
    sys.exit("no previous release tag found; pass --previous-version")


def github_login(repo, sha, token):
    url = "https://api.github.com/repos/{}/commits/{}".format(repo, sha)
    headers = {"Accept": "application/vnd.github+json"}
    if token:
        headers["Authorization"] = "Bearer " + token
    try:
        req = urllib.request.Request(url, headers=headers)
        with urllib.request.urlopen(req, timeout=30) as resp:
            body = json.loads(resp.read())
        return (body.get("author") or {}).get("login")
    except (urllib.error.URLError, json.JSONDecodeError) as err:
        print("warning: no login for {}: {}".format(sha[:8], err), file=sys.stderr)
        return None


def changelog_entries(repo, since, token):
    log = run(
        "git", "log", "{}..HEAD".format(since), "--no-merges", "--pretty=%H\x1f%s\x1f%an"
    )
    entries, skipped = [], []
    for line in log.split("\n"):
        if not line.strip():
            continue
        sha, subject, author = line.split("\x1f")
        if SKIP_SUBJECT_RE.search(subject):
            skipped.append(subject)
            continue
        login = github_login(repo, sha, token) or author
        if LOGIN_RE.match(login):
            credit = "[@{0}](https://github.com/{0})".format(login)
        else:
            # A git author name, not a login: linking it would 404. This happens
            # for commits that are not pushed to GitHub yet.
            credit = "@{}".format(login)
            print(
                "warning: {} has no GitHub login, crediting '{}' without a link".format(
                    sha[:8], login
                ),
                file=sys.stderr,
            )
        short = sha[:8]
        entries.append(
            "* {} ([#{}](https://github.com/{}/commit/{}), {})".format(
                subject, short, repo, short, credit
            )
        )
    for subject in skipped:
        print("skipping release commit: {}".format(subject), file=sys.stderr)
    return entries


def update_changelog(version, entries, dry_run):
    body = open(CHANGELOG).read()
    if body.startswith("# {}\n".format(version)):
        print("{}: already has a {} section".format(CHANGELOG, version))
        return
    section = "# {}\n{}\n\n".format(version, "\n".join(entries))
    print("{}: adding {} ({} entries)".format(CHANGELOG, version, len(entries)))
    if not dry_run:
        open(CHANGELOG, "w").write(section + body)


def update_makefile(version, dry_run):
    body = open(MAKEFILE).read()
    new, n = re.subn(
        r"^VERSION\?=\S+$", "VERSION?={}".format(version), body, count=1, flags=re.M
    )
    if n != 1:
        sys.exit("{}: could not find the VERSION?= line".format(MAKEFILE))
    if new == body:
        print("{}: already at {}".format(MAKEFILE, version))
        return
    print("{}: VERSION?={}".format(MAKEFILE, version))
    if not dry_run:
        open(MAKEFILE, "w").write(new)


def update_install(branch, dry_run):
    body = open(INSTALL).read()
    new, n = re.subn(r"\?ref=release-[\d.]+", "?ref={}".format(branch), body)
    if n == 0:
        sys.exit("{}: could not find a ?ref=release-* pointer".format(INSTALL))
    if new == body:
        print("{}: already points at {}".format(INSTALL, branch))
        return
    print("{}: ?ref={} ({} occurrence{})".format(INSTALL, branch, n, "" if n == 1 else "s"))
    if not dry_run:
        open(INSTALL, "w").write(new)


def new_row(template, version, kind):
    """Build an aligned table row, taking column widths from the row below it."""
    cells = template.split("|")[1:-1]
    if kind == "image":
        values = [version, "{}:{}".format(IMAGE_REPO, version)]
    else:
        values = [version] + [c.strip() for c in cells[1:]]
    padded = [
        " {} ".format(v.ljust(max(len(c) - 2, len(v)))) for v, c in zip(values, cells)
    ]
    return "|" + "|".join(padded) + "|"


def update_readme(version, dry_run):
    lines = open(README).read().split("\n")
    out, added = [], 0
    i = 0
    while i < len(lines):
        out.append(lines[i])
        heading = next((h for h, _ in MATRICES if lines[i].strip() == h), None)
        if heading:
            kind = dict(MATRICES)[heading]
            # Copy the table header and separator across, then insert our row.
            j = i + 1
            while j < len(lines) and not SEPARATOR_RE.match(lines[j]):
                out.append(lines[j])
                j += 1
            if j >= len(lines):
                sys.exit("{}: no table under {}".format(README, heading))
            out.append(lines[j])  # the |---| separator
            template = lines[j + 1]
            if template.split("|")[1].strip() == version:
                print("{}: {} already lists {}".format(README, heading, version))
            else:
                out.append(new_row(template, version, kind))
                added += 1
            i = j
        i += 1
    if added:
        print("{}: added {} row{}".format(README, added, "" if added == 1 else "s"))
        if not dry_run:
            open(README, "w").write("\n".join(out))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--version", required=True, help="version to release, e.g. v1.12.0")
    parser.add_argument("--previous-version", help="defaults to the newest vX.Y.Z tag")
    parser.add_argument(
        "--release-branch",
        help="branch docs/install.md should point ?ref= at (default release-X.Y)",
    )
    parser.add_argument("--repo", default="kubernetes-sigs/aws-fsx-csi-driver")
    parser.add_argument("--dry-run", action="store_true", help="report without writing")
    parser.add_argument("--summary-file", help="write the changelog section here")
    args = parser.parse_args()

    if not VERSION_RE.match(args.version):
        sys.exit("--version must look like v1.12.0, got {}".format(args.version))
    for path in (CHANGELOG, MAKEFILE, README, INSTALL):
        if not os.path.exists(path):
            sys.exit("run me from the repository root; missing {}".format(path))

    major, minor, _ = VERSION_RE.match(args.version).groups()
    branch = args.release_branch or "release-{}.{}".format(major, minor)
    since = args.previous_version or previous_version(args.version)
    print("preparing {} (since {}), release branch {}\n".format(args.version, since, branch))

    entries = changelog_entries(args.repo, since, os.environ.get("GITHUB_TOKEN"))
    if not entries:
        sys.exit("no commits between {} and HEAD; nothing to release".format(since))

    update_changelog(args.version, entries, args.dry_run)
    update_makefile(args.version, args.dry_run)
    update_readme(args.version, args.dry_run)
    update_install(branch, args.dry_run)

    if args.summary_file:
        with open(args.summary_file, "w") as fh:
            fh.write("\n".join(entries) + "\n")

    print("\ndone{}".format(" (dry run)" if args.dry_run else ""))


if __name__ == "__main__":
    main()
