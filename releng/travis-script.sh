#!/usr/bin/env bash
set -o errexit
set -o nounset

git_branch="$1"
git_commit="$2"

printf "== Begin: travis-script.sh (branch: $git_branch, commit: $git_commit) ==\n"

printf "== End:   travis-script.sh ==\n"
