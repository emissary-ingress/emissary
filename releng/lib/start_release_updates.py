import fileinput
import sys
from lib import git_add
import re

def update_changelog_date(next_ver):
    changelog_ver_pattern = re.compile(r"^## \[([0-9]+\.[0-9]+\.[0-9]+(-ea)?)\]")
    in_notes = False
    buf = ""
    for line in fileinput.FileInput("CHANGELOG.md", inplace=True):
        if not in_notes:
            sys.stdout.write(line)
            if line.startswith("## Next Release"):
                in_notes = True
            continue

        match = changelog_ver_pattern.match(line)
        if not match:
            buf += line
        elif line.startswith(f"## [{next_ver}]"):
            # Don't do anything, this changelog already has an entry for the next version
            sys.stdout.write(buf)
            sys.stdout.write(line)
            in_notes = False
        else:
            prev_ver = match[1]
            # dope let's get the last version first
            # this is the beginning of the last version line
            sys.stdout.write("\n")
            sys.stdout.write("### Emissary Ingress\n")
            sys.stdout.write("\n")
            sys.stdout.write("(no changes yet)\n")
            sys.stdout.write("\n")
            sys.stdout.write(f"## [{next_ver}] (TBD)\n")
            sys.stdout.write(
                    f"[{next_ver}]: https://github.com/emissary-ingress/emissary/compare/v{prev_ver}...v{next_ver}\n")
            sys.stdout.write(buf)
            sys.stdout.write(line)
            in_notes = False

    git_add("CHANGELOG.md")
