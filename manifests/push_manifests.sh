#!/bin/bash

set -e -o pipefail

RED=$'\033[1;31m'
END=$'\033[0m'
BLOCK=$'\033[1;47m'

abort() {
    log "${RED}FATAL: $1${END}"
    exit 1
}

log() { >&2 printf '%s>>>%s %s\n' "$BLOCK" "$END" "$*"; }

errusage() {
    printf >&2 'Usage: %s DIR\n' "$0"
    if [[ $# -gt 0 ]]; then
        local msg
        # shellcheck disable=SC2059
        printf -v msg "$@"
        printf >&2 '%s: error: %s\n' "$0" "$msg"
    fi
    exit 2
}

[[ $# == 1                     ]] || errusage 'wrong number of args: %d' $#
[[ -d "$1"                     ]] || errusage 'DIR is not a directory: %q' "$dir"
[[ -n "$AWS_ACCESS_KEY_ID"     ]] || errusage "AWS_ACCESS_KEY_ID is not set"
[[ -n "$AWS_SECRET_ACCESS_KEY" ]] || errusage "AWS_SECRET_ACCESS_KEY is not set"
[[ "${VERSION:-}" == v3.*      ]] || errusage "VERSION must be set to a 'v3.*' string"
dir=$1
while [[ "$dir" == */ ]]; do
    dir=${dir%/}
done
version=${VERSION#v}

log "Publishing manifest version ${version}"

if [[ $version =~ ^[0-9]+\.[0-9]+\.[0-9]+(-ea)?$ ]] ; then
    # if this is a stable version, working directory must be clean
    # otherwise this is an rc or test version and we don't care
    if [ -n "$(git status --porcelain)" ] ; then
        abort "working tree is dirty, aborting"
    fi
elif [[ "${BUMP_STABLE}" ]] ; then
    # if this isn't an X.Y.Z version, don't let allow bumping stable
    abort "Cannot bump stable unless this is an X.Y.Z tag"
fi
if [ -z "$AWS_S3_BUCKET" ] ; then
    AWS_S3_BUCKET=datawire-static-files
fi

echo "$version" > stable.txt

aws s3api put-object \
    --bucket "$AWS_S3_BUCKET" \
    --key "yaml/emissary/${version}"

find "$dir" -type f -name '*.yaml' -print0 | while read -r -d '' file; do
    aws s3api put-object \
        --bucket "$AWS_S3_BUCKET" \
        --key "yaml/emissary/${version}/${file##*/}" \
        --body "$file"
    echo "... yaml/emissary/${version}/${file##*/} pushed"
done

if [[ "${BUMP_STABLE}" ]] ; then
    log "Bumping stable version for yaml/emissary"
    aws s3api put-object \
        --bucket "$AWS_S3_BUCKET" \
        --key "yaml/emissary/stable.txt" \
        --body stable.txt
fi
rm stable.txt

exit 0
