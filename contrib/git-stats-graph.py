#!/usr/bin/env python3
# SPDX-License-Identifier: MIT
# Copyright (c) 2023 Bence Ferdinandy <bence@ferdinandy.com>

"""
Create graphs about development statistics of releases.
"""

from datetime import date
from subprocess import check_output

from matplotlib import pyplot as plt


def git(*args):
    return check_output(["git"] + list(args)).decode("utf-8").strip()


def stats():
    """
    Returns statistics from the git repo:

    tags: sorted list of minor version tags (assumes semver)
          The first element is the hash of the first commit, last element is HEAD.  All
          the other return values are one shorter as, there's no statistics returned for
          the first commit.
    counts: number of commits (between this and the previous release)
    dates: dates of the releases
    files: number of files changed (between this and the previous release)
    inserts: number of lines inserted (between this and the previous release)
    deletions: number of lines deleted (between this and the previous release)
    """

    tags = git("tag").split("\n")
    tags = [t for t in tags if t.split(".")[-1] == "0"]  # drop patch versions
    tags = sorted(tags, key=lambda x: [int(t) for t in x.split(".")])
    first_commit = git("rev-list", "--max-parents=0", "HEAD")
    tags = [first_commit] + tags + ["HEAD"]
    counts = []
    dates = []
    files = []
    inserts = []
    deletions = []
    for i, t in enumerate(tags[:-1]):
        counts.append(int(git("rev-list", f"{t}..{tags[i+1]}", "--count")))
        dates.append(
            date.fromisoformat(
                git("show", "-s", "--format=%cs", tags[i + 1]).split("\n")[-1]
            )
        )
        statline = git("diff", "--stat", t, tags[i + 1]).split("\n")[-1]
        fnum, _, _, ins, _, dels, _ = statline.split()
        files.append(int(fnum))
        inserts.append(int(ins))
        deletions.append(int(dels))
    return tags, counts, dates, files, inserts, deletions


def main(output):
    tags, counts, dates, files, inserts, deletions = stats()

    fig, (ax1, ax2) = plt.subplots(2, figsize=(8, 11))
    fig.suptitle("aerc release statistics", fontweight="bold")
    # commit counts subplot
    ax1.plot(dates, counts, "o-")

    # alternate placement of text above and below for readability
    text_y = []
    for i, t in enumerate(tags[1:]):
        downpad = 25 if len(t) == 5 else 30
        p = counts[i] + (-1) ** (i + 1) * 10 - (i + 1) % 2 * downpad
        if p < 5:
            p = counts[i] + 10
        text_y.append(p)
    for i, t in enumerate(tags[1:]):
        ax1.text(
            dates[i],
            text_y[i],
            t,
            horizontalalignment="center",
            rotation="vertical",
        )
    ax1.set_ylabel("# of commits")
    ax1.set_ylim(bottom=0)
    ax1.set_title("commits per release")

    # lines added/deleted subplot
    #
    ax2.plot(dates, inserts, "o-", label="insertions(+)", color="green")
    ax2.plot(dates, deletions, "o-.", label="deletions(-)", color="red")
    ax2.set_ylabel("# of lines")
    ax2.legend(loc="upper left")
    ax2.set_ylim(top=max(max(inserts), max(deletions)) + 2000)
    for i, t in enumerate(tags[1:]):
        ax2.text(
            dates[i],
            max(inserts[i], deletions[i]) + 500,
            t,
            horizontalalignment="center",
            rotation="vertical",
        )
    ax2.set_xlabel("date")
    ax2.set_title("insertion/deletions per release")
    plt.tight_layout()
    plt.savefig(output, dpi=300)


if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "-o",
        "--output",
        default="aerc-release-stats.png",
        help="""
        Path to output image (defaults to 'aerc-release-stats.png',
        respects file extensions via matplotlib)
        """,
    )
    args = parser.parse_args()
    main(args.output)
