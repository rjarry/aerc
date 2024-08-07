#!/usr/bin/env python3
# SPDX-License-Identifier: MIT
# Copyright (c) 2024 Robin Jarry

import argparse
import re
import subprocess

DEP_CHANGE_RE = re.compile(
    r"""
    ^
    (?P<diff>[\+\-])\s*
    (?P<name>\S+)\s*
    (?P<version>v\S+)\s*
    (?://\s*indirect)?
    $
    """,
    re.VERBOSE,
)
REPLACE_RE = re.compile(
    r"""
    ^
    (?P<diff>[\+\-])\s*
    replace
    (?P<name>\S+)\s*
    =>\s*
    (?P<replacement>\S+)\s*
    (?P<version>v\S+)\s*
    $
    """,
    re.VERBOSE,
)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "git_range",
        metavar="GIT_RANGE",
        help="The git revision range (see gitrevisions(7)).",
    )
    args = parser.parse_args()

    old_deps = {}
    new_deps = {}

    with subprocess.Popen(
        ["git", "diff", "-U0", "--ignore-all-space", args.git_range, "--", "go.mod"],
        stdout=subprocess.PIPE,
        encoding="utf-8",
    ) as proc:
        for line in proc.stdout:
            match = DEP_CHANGE_RE.match(line.strip())
            if not match:
                match = REPLACE_RE.match(line.strip())
                if not match:
                    continue
                diff, name, replacement, version = match.groups()
                if diff == "+":
                    new_deps[replacement] = version
                    del new_deps[name]
                continue
            diff, name, version = match.groups()
            if diff == "+":
                new_deps[name] = version
            else:
                old_deps[name] = version

    once = False
    added = new_deps.keys() - old_deps.keys()
    if added:
        print("## New")
        print()
        for a in sorted(added):
            print("+", a, new_deps[a])
        once = True

    updated = old_deps.keys() & new_deps.keys()
    if updated:
        if once:
            print()
        print("## Updated")
        print()
        for u in sorted(updated):
            print("*", u, old_deps[u], "=>", new_deps[u])
        once = True

    removed = old_deps.keys() - new_deps.keys()
    if removed:
        if once:
            print()
        print("## Removed")
        print()
        for r in sorted(removed):
            print("-", r)
        once = True

    if not once:
        print("none")


if __name__ == "__main__":
    main()
