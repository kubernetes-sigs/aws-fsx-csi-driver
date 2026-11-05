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

"""Move the Go toolchain to the latest stable release.

The builder image in the Dockerfile tracks the exact patch release. The `go`
directives in the two go.mod files track the minor only (go 1.27.0), which is
how this repository has always pinned them: the directive is the floor we ask
consumers to have, so it moves when the minor moves and not on every patch.
"""

import argparse
import json
import os
import re
import sys
import urllib.request

DOCKERFILE = "Dockerfile"
GOMODS = ["go.mod", "tests/e2e/go.mod"]
README = "docs/README.md"

BUILDER_RE = re.compile(r"^(FROM .*golang:)(\d+\.\d+(?:\.\d+)?)( .*)$", re.M)
DIRECTIVE_RE = re.compile(r"^go (\d+\.\d+\.\d+)$", re.M)
REQUIREMENT_RE = re.compile(r"^\* Golang (\d+\.\d+\.\d+)\+$", re.M)


def latest_stable():
    with urllib.request.urlopen("https://go.dev/dl/?mode=json", timeout=30) as resp:
        releases = json.loads(resp.read())
    for release in releases:
        if release.get("stable") and release["version"].startswith("go"):
            return release["version"][2:]
    sys.exit("go.dev/dl listed no stable release")


def patch_file(path, pattern, replacement, label, dry_run):
    body = open(path).read()
    new, n = pattern.subn(replacement, body)
    if n == 0:
        sys.exit("{}: could not find {}".format(path, label))
    if new == body:
        print("{}: {} already current".format(path, label))
        return False
    print("{}: {}".format(path, label))
    if not dry_run:
        open(path, "w").write(new)
    return True


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--go-version", help="override the target version, e.g. 1.27.2")
    parser.add_argument("--dry-run", action="store_true", help="report without writing")
    args = parser.parse_args()

    for path in [DOCKERFILE, README] + GOMODS:
        if not os.path.exists(path):
            sys.exit("run me from the repository root; missing {}".format(path))

    version = args.go_version or latest_stable()
    if not re.match(r"^\d+\.\d+\.\d+$", version):
        sys.exit("--go-version must look like 1.27.2, got {}".format(version))
    major, minor, _ = version.split(".")
    floor = "{}.{}.0".format(major, minor)

    print("latest stable Go: {} (module floor {})\n".format(version, floor))

    changed = patch_file(
        DOCKERFILE,
        BUILDER_RE,
        lambda m: m.group(1) + version + m.group(3),
        "builder image golang:{}".format(version),
        args.dry_run,
    )
    for path in GOMODS:
        changed |= patch_file(
            path, DIRECTIVE_RE, "go " + floor, "go " + floor, args.dry_run
        )
    changed |= patch_file(
        README,
        REQUIREMENT_RE,
        "* Golang {}+".format(floor),
        "Golang {}+".format(floor),
        args.dry_run,
    )

    if os.environ.get("GITHUB_OUTPUT"):
        with open(os.environ["GITHUB_OUTPUT"], "a") as fh:
            fh.write("go_version={}\n".format(version))
            fh.write("changed={}\n".format("true" if changed else "false"))

    print("\n{}{}".format(
        "updated" if changed else "nothing to do",
        " (dry run)" if args.dry_run else "",
    ))


if __name__ == "__main__":
    main()
