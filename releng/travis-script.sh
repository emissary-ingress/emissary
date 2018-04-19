#!/usr/bin/env bash
set -o errexit
set -o nounset

git_branch="$1"
git_commit="$2"
version=${3:-$git_commit}

printf "== Begin: travis-script.sh (branch: $git_branch, commit: $git_commit, version: $version) ==\n"

make clean

printf "== Begin: execute tests\n"

make test

printf "== End:   execute tests\n"

printf "== Begin: build docker image\n"

make docker-images

printf "== End:   build docker image\n"

printf "== Begin: generate documentation\n"

make website

printf "== End:   generate documentation\n"

printf "== End:   travis-script.sh ==\n"
