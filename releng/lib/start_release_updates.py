import fileinput
import sys
from lib import git_add
import re

def update_versions_yaml(next_ver):
    for line in fileinput.FileInput("docs/yaml/versions.yml", inplace=True):
        if line.startswith("version:"):
            line = f"version: {next_ver}\n"
        sys.stdout.write(line)
    git_add("docs/yaml/versions.yml")

def update_changelog_date(next_ver):
    changelog_ver_pattern = re.compile(r"^## \[([0-9]+\.[0-9]+\.[0-9]+(-ea)?)\]")
    in_notes = False
    buf = ""
    found = False
    for line in fileinput.FileInput("CHANGELOG.md", inplace=True):
        if not in_notes:
            sys.stdout.write(line)
            if line.startswith("## RELEASE NOTES"):
                in_notes = True
            continue
        if not found and line.startswith('## Next Release'):
            found = True
            sys.stdout.write(line)
            sys.stdout.write('\n\n### Emissary Ingress\n\n')
            sys.stdout.write('(no changes yet)\n\n')
            sys.stdout.write(f"## [{next_ver}] (TBD)\n")
            sys.stdout.write(
                    f"[{next_ver}]: https://github.com/emissary-ingress/emissary/compare/v{prev_ver}...v{next_ver}\n")
        else:
            sys.stdout.write(line)
    git_add("CHANGELOG.md")
