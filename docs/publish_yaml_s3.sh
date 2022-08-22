#!/bin/bash

set -e -o pipefail

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

if [[ $version =~ ^[0-9]+\.[0-9]+\.[0-9]+(-ea)?$ ]] ; then
    # if this is a stable version, working directory must be clean
    # otherwise this is an rc, ea or test version and we don't care
    if [ -n "$(git status --porcelain)" ] ; then
        echo "working tree is dirty, aborting"
        exit 1
    fi
elif [[ "$BUMP_STABLE" = "true" ]] ; then
    # if this isn't an X.Y.Z version, don't let allow bumping stable
    echo "Cannot bump stable unless this is an X.Y.Z tag"
    exit 1
fi

echo "$version" > stable.txt
if [ -z "$AWS_S3_BUCKET" ] ; then
    AWS_S3_BUCKET=datawire-static-files
fi

# make this something different than ambassador, emissary, or edge-stack
# so we don't conflict with the new hotness we're doing for 2.0
unversioned_base_s3_key=yaml/v2-docs/
base_s3_key=${unversioned_base_s3_key}${version}
aws s3api put-object \
    --bucket "$AWS_S3_BUCKET" \
    --key "${base_s3_key}"

echo "Pushing files to s3..."
find "$dir" -type f -print0 | while read -r -d '' file; do
    s3_key=${base_s3_key}/${file#"${dir}/"}
    aws s3api put-object \
        --bucket "$AWS_S3_BUCKET" \
        --key "${s3_key}" \
        --body "$file"
    echo "... ${s3_key} pushed"
done

if [[ "${BUMP_STABLE}" = "true" ]] ; then
    echo "Bumping stable version for yaml/${dir}"
    aws s3api put-object \
        --bucket "$AWS_S3_BUCKET" \
        --key "${unversioned_base_s3_key}stable.txt" \
        --body stable.txt
fi

echo "Done pushing files to s3"
rm stable.txt
