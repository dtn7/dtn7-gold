#!/usr/bin/env python

# SPDX-FileCopyrightText: 2020 Alvar Penning
#
# SPDX-License-Identifier: GPL-3.0-or-later

import subprocess

from datetime import datetime
from typing import List, Tuple


# License to be applied.
DEFAULT_LICENSE = "GPL-3.0-or-later"

# Rename authors, the following happened due to using GitHub's Web UI.
AUTHOR_REWRITE = {"Alvar": "Alvar Penning"}

# Exclude unsupported files.
FILE_IGNORE = [
        "go.mod", "go.sum",  # https://github.com/fsfe/reuse-tool/pull/234
        ".reuse/dep5",
        "LICENSE", "LICENSES/CC0-1.0.txt", "LICENSES/GPL-3.0-or-later.txt"
        ]


# Ownership is a triple of (Filename, Author, Years)
Ownership = Tuple[str, str, str]


def git(cmds: List[str]) -> str:
    "Execute a git subcommand, returning stdout as a string."

    proc = subprocess.run(["git"] + cmds,
            capture_output=True, text=True, check=True)
    return proc.stdout


def file_authors(f: str) -> List[Ownership]:
    "List all ownerships of the requested file."

    owners_raw = git(["log", "--pretty=format:%an%x00%at", f]).split("\n")
    owners_tpl = [raw.split(chr(0)) for raw in owners_raw]
    owners_tpl.reverse()

    owners_year = [(tpl[0], datetime.fromtimestamp(int(tpl[1])).strftime("%Y"))
            for tpl in owners_tpl]

    owner_grp = dict()
    for author, year in set(owners_year):
        author = AUTHOR_REWRITE.get(author, author)

        if not author in owner_grp:
            owner_grp[author] = [year]
        elif year not in owner_grp[author]:
            owner_grp[author].append(year)
            owner_grp[author].sort()

    owners = [(f, author, ", ".join(years))
            for (author, years) in owner_grp.items()]
    sorted(owners, key=lambda owner: owner[1], reverse=True)

    return owners


def reuse_addheader(f: str, author: str, years: str, license: str):
    "Execute `reuse addheader`."

    subprocess.run(["reuse", "addheader",
        f"--copyright={author}", f"--year={years}", f"--license={license}", f],
        check=True)


if __name__ == "__main__":
    files = filter(lambda f: f not in FILE_IGNORE, git(["ls-files"]).split())

    owners = []
    for f in files:
        owners = owners + file_authors(f)

    for f, author, years in owners:
        reuse_addheader(f, author, years, DEFAULT_LICENSE)
