#!/bin/bash

set -ex

CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir (maybe running in zsh?)";  exit 1; }
RED='\033[1;31m'
END='\033[0m'
BLOCK='\033[1;47m'

log() { >&2 printf "${BLOCK}>>>${END} $1\n"; }

abort() {
  log "${RED}FATAL: $1${END}"
  exit 1
}

[ -n "$MANIFEST_VERSION"     ] || abort "MANIFEST_VERSION is not set"

if [ -z "$AWS_BUCKET" ] ; then
    AWS_BUCKET=datawire-static-files
fi

[ -n "$AWS_ACCESS_KEY_ID"     ] || abort "AWS_ACCESS_KEY_ID is not set"
[ -n "$AWS_SECRET_ACCESS_KEY" ] || abort "AWS_SECRET_ACCESS_KEY is not set"

if [ -z "${FORCE}" ] ; then
    if aws s3api get-object --bucket ${AWS_BUCKET} --key emissary-yaml/${MANIFEST_VERSION} /dev/null ; then
        abort "FORCE is not set and manifests already exist for ${MANIFEST_VERSION}"
    fi
fi
# TODO: check if there's already a directory for the manifest versions,
# abort if things exists and force isn't set
# and probably if git ref isn't up to date with manifest version?
aws s3api put-object \
    --bucket "$AWS_BUCKET" \
    --key "emissary-yaml/${MANIFEST_VERSION}"

log "Pushing manifests to S3 bucket $AWS_BUCKET"
for f in ${CURR_DIR}/*.yaml ; do
  fname=`basename $f`
  aws s3api put-object \
    --bucket "$AWS_BUCKET" \
    --key "emissary-yaml/${MANIFEST_VERSION}/$fname" \
    --body "$f" &&  echo "... emisasry-yaml/$fname pushed"
done

exit 0
