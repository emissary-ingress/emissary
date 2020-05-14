#!/usr/bin/env bash

# Copyright 2018 Datawire. All rights reserved.
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
# limitations under the License

set -e

press_enter() {
    read -s -p "Press enter to continue"
    echo
}

RELEASE_PROMPT="
Before making the release, make sure that -
- [ ] the new version number depicts the release accurately
- [ ] if new features have been added, the MINOR version is bumped
- [ ] if backwards-incompatible changes have been made, the MINOR version is bumped
- [ ] if only backwards-compatible bug fixes have been made, the PATCH version is bumped
- [ ] the changes and the version number has been vetted by project maintainers and stakeholders
"

echo "$RELEASE_PROMPT"
press_enter

echo "Fetching latest release tag from GitHub"
CURRENT_VERSION=$(curl --silent "https://api.github.com/repos/datawire/ambassador/releases/latest" | fgrep 'tag_name' | cut -d'"' -f 4)
echo
echo "Current version: ${CURRENT_VERSION}"
echo
git log --pretty=oneline --abbrev-commit ${CURRENT_VERSION}^..
echo
echo "^ these changes have been made since the last release, pick the right version number accordingly"

while true; do
	read -p "Enter new version: " DESIRED_VERSION

	if [[ "$DESIRED_VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+-rc\.[0-9]+$ ]]; then
	    # RC: good.
	    break
	elif [[ "$DESIRED_VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+-ea\.[0-9]+$ ]]; then
	    # EA: good.
	    break
	else
	    echo "'$DESIRED_VERSION' is not in one of the recognized tag formats:" >&2
	    echo " - 'SEMVER-rc.N'" >&2
	    echo " - 'SEMVER-ea.N'" >&2
	    echo "Note that the tag name must not start with 'v'" >&2
	fi
done

echo "Desired version: ${DESIRED_VERSION}"
press_enter
echo

# # Update version number in docs/index.html
# echo "Found the following occurrences of $CURRENT_VERSION in ./docs/index.html"
# echo
# grep -C 2 ${CURRENT_VERSION} docs/index.html
# echo

# echo "Replacing $CURRENT_VERSION with $DESIRED_VERSION in ./docs/index.html"
# sed -i -e "s/$CURRENT_VERSION/$DESIRED_VERSION/g" docs/index.html
# echo
# git diff docs/index.html
# echo
# echo "^ this is the 'git diff' for docs/index.html"
# press_enter

echo "Updating version to $DESIRED_VERSION in ./docs/versions.yml"
sed -i -e "s/^version: .*$/version: $DESIRED_VERSION/g" docs/versions.yml
echo
git diff docs/versions.yml
echo
echo "^ this is the 'git diff' for docs/versions.yml"
press_enter

# Update release notes and changelog now
RELEASE_NOTES_TEMPLATE="Delete everything from this template that you do not want to be in the template, including this line.

### Major changes:
- Feature: <insert feature description here>
- Bugfix: <insert bugfix description here>

### Minor changes:
- Feature: <insert feature description here>
- Bugfix: <insert bugfix description here>"

temp_file=$(mktemp)
echo "${RELEASE_NOTES_TEMPLATE}" > ${temp_file}

if ! ${EDITOR:-vi} ${temp_file}; then
    exit 1
fi

RELEASE_NOTES=$(<${temp_file})
trap 'rm -f ${temp_file}' EXIT

current_v=$CURRENT_VERSION

if [[ "$current_v" != v* ]]; then
	current_v="v${CURRENT_VERSION}"
fi

desired_v=$DESIRED_VERSION

if [[ "$desired_v" != v* ]]; then
	desired_v="v${DESIRED_VERSION}"
fi

CHANGELOG="## [${DESIRED_VERSION}] $(date "+%B %d, %Y")
[${DESIRED_VERSION}]: https://github.com/datawire/ambassador/compare/${current_v}...${desired_v}

${RELEASE_NOTES}
"

echo ""
echo "====================================="
echo "Generated changelog -"
echo "${CHANGELOG}"
press_enter
echo "Updating CHANGELOG.md..."
echo "${CHANGELOG}" | sed -i -e '/CueAddReleaseNotes/r /dev/stdin' CHANGELOG.md
echo "...done. Diffs for CHANGELOG.md:"
git diff CHANGELOG.md
echo
echo "^ this is the 'git diff' for CHANGELOG.md"
press_enter

echo ""
echo "====================================="
echo "Final diffs to be committed:"
git diff docs/versions.yml CHANGELOG.md
echo
echo "^ This is final 'git diff'. Bail out now if you do not want to commit this."
press_enter

echo "Staging docs/versions.yml and CHANGELOG.md"
git add docs/versions.yml CHANGELOG.md
echo "Committing docs/versions.yml and CHANGELOG.md"
git commit -m "Ambassador ${DESIRED_VERSION} release" docs/versions.yml CHANGELOG.md

GITHUB_RELEASE_NOTES="## :tada: Ambassador ${DESIRED_VERSION} :tada:
#### Ambassador is an open source, Kubernetes-native microservices API gateway built on the Envoy Proxy.

Upgrade Ambassador - https://www.getambassador.io/reference/upgrading.html
View changelog - https://github.com/datawire/ambassador/blob/master/CHANGELOG.md
Get started with Ambassador on Kubernetes - https://www.getambassador.io/user-guide/getting-started

${RELEASE_NOTES}"

echo ""
echo "====================================="
echo "Paste the following content in GitHub release page once the release is done -"
echo
echo "${GITHUB_RELEASE_NOTES}"