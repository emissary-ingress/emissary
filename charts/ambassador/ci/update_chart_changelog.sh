#!/bin/bash

set -e

CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir (maybe running in zsh?)";  exit 1; }
TOP_DIR=$CURR_DIR/..

# shellcheck source=common.sh
source "$CURR_DIR/common.sh"

chart_version=$(get_chart_version ${TOP_DIR})

new_changelog=${TOP_DIR}/CHANGELOG.new.md
rm ${new_changelog} || true
while IFS= read -r line ; do
    echo -e "${line}"
    echo -e "${line}" >> ${new_changelog}
    if [[ "${line}" =~ "## Next Release" ]] ; then
        echo "" >> ${new_changelog}
        echo "(no changes yet)" >> ${new_changelog}
        echo "" >> ${new_changelog}
        echo "## v${chart_version}" >> ${new_changelog}
    fi

done < ${TOP_DIR}/CHANGELOG.md

mv ${new_changelog} ${TOP_DIR}/CHANGELOG.md
if [[ -n "${DONT_COMMIT_DIFF}" ]] ; then
    echo "DONT_COMMIT_DIFF is set, not committing"
    exit 0
fi

if git diff --exit-code -- ${TOP_DIR}/CHANGELOG.md > /dev/null 2>&1 ; then
    echo "No changes to changelog, exiting"
    exit 0
fi

branch_name="$(git symbolic-ref HEAD 2>/dev/null)" ||
branch_name="detached"

if [[ "${branch_name}" == "refs/heads/master" ]] ; then
    echo "Not committing local changes to branch because branch is master"
    exit 1
elif [[ "${branch_name}" == "detached" ]] ; then
    echo "Not committing local changes because you're in a detached head state"
    echo "please create a branch then rerun this script"
    exit 1
fi
branch_name=${branch_name##refs/heads/}
git add ${TOP_DIR}/CHANGELOG.md
git commit -m "Committing changelog for chart v${chart_version}"
git push -u origin ${branch_name}
