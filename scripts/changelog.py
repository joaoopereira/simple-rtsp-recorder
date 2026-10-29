#!/usr/bin/env python3
"""Generates CHANGELOG.md sections from conventional commits.

Keeps the format commit-and-tag-version used for 1.x: features, bug fixes
and breaking changes are listed; chore/ci/docs/test commits are left out.

  scripts/changelog.py v2.0.4                      # section for HEAD since the last tag
  scripts/changelog.py v2.0.4 --update CHANGELOG.md  # also insert it into the file
  scripts/changelog.py v2.0.1 --from v2.0.0 --to v2.0.1  # an older range

Prints the section to stdout (used as the GitHub release notes).
Standard library only.
"""
import argparse
import re
import subprocess
import sys

REPO_URL = "https://github.com/joaoopereira/simple-rtsp-recorder"
SECTIONS = [("feat", "Features"), ("fix", "Bug Fixes"), ("perf", "Performance Improvements")]
SUBJECT = re.compile(r"^(?P<type>\w+)(?:\((?P<scope>[^)]*)\))?(?P<bang>!)?: (?P<desc>.+)$")


def git(*args):
    return subprocess.run(["git", *args], check=True, capture_output=True, text=True).stdout.strip()


def commits(rev_range):
    # One record per commit: hash, subject, body; separated by control chars.
    out = git("log", "--no-merges", "--format=%H%x1f%s%x1f%b%x1e", rev_range)
    for record in out.split("\x1e"):
        if record.strip():
            sha, subject, body = (record.strip("\n").split("\x1f") + ["", ""])[:3]
            yield sha, subject, body


def section(version, from_ref, to_ref, date):
    entries = {t: [] for t, _ in SECTIONS}
    breaking = []
    for sha, subject, body in commits(f"{from_ref}..{to_ref}"):
        m = SUBJECT.match(subject)
        if not m:
            continue
        scope = f"**{m['scope']}:** " if m["scope"] else ""
        link = f"([{sha[:7]}]({REPO_URL}/commit/{sha}))"
        if m["type"] in entries:
            entries[m["type"]].append(f"* {scope}{m['desc']} {link}")
        note = re.search(r"^BREAKING[ -]CHANGE: (.+?)(?:\n\n|\Z)", body, re.M | re.S)
        if note:
            breaking.append(f"* {scope}{' '.join(note.group(1).split())}")
        elif m["bang"]:
            breaking.append(f"* {scope}{m['desc']}")

    number = version.lstrip("v")
    out = [f"## [{number}]({REPO_URL}/compare/{from_ref}...{version}) ({date})", ""]
    if breaking:
        out += ["", "### ⚠ BREAKING CHANGES", ""] + breaking
    for key, title in SECTIONS:
        if entries[key]:
            out += ["", f"### {title}", ""] + entries[key]
    return "\n".join(out).rstrip() + "\n"


def insert(path, text):
    with open(path, encoding="utf-8") as f:
        content = f.read()
    if text.split("]", 1)[0] + "]" in content:
        return  # already there, e.g. a re-run of a failed release
    marker = content.find("\n## [")
    if marker < 0:
        content = content.rstrip() + "\n\n" + text
    else:
        content = content[: marker + 1] + text + "\n" + content[marker + 1 :]
    with open(path, "w", encoding="utf-8") as f:
        f.write(content)


def main():
    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    p.add_argument("version", help="version being released, e.g. v2.0.4")
    p.add_argument("--from", dest="from_ref", help="previous tag (default: latest v* tag before --to)")
    p.add_argument("--to", dest="to_ref", default="HEAD")
    p.add_argument("--date", help="YYYY-MM-DD (default: date of --to)")
    p.add_argument("--update", metavar="FILE", help="insert the section into this changelog")
    a = p.parse_args()

    # Latest tag at or before --to, other than the one being generated.
    from_ref = a.from_ref or git("describe", "--tags", "--abbrev=0", "--match", "v*", "--exclude", a.version, a.to_ref)
    date = a.date or git("log", "-1", "--format=%cs", a.to_ref)
    text = section(a.version, from_ref, a.to_ref, date)
    if a.update:
        insert(a.update, text)
    sys.stdout.write(text)


if __name__ == "__main__":
    main()
