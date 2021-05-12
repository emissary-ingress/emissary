#!/bin/bash

set -e

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

TOP_DIR=${CURR_DIR}/../
version=$(grep version ${TOP_DIR}docs/yaml/versions.yml | awk '{ print $2 }')
[ -n ${version} ] || abort "could not read version from docs/yaml/versions.yml"
log "Publishing manifest version ${version}"

if [ -n "$(git status --porcelain)" ]; then
    abort "working tree is dirty, aborting"
fi
if [ -z "$AWS_BUCKET" ] ; then
    AWS_BUCKET=datawire-static-files
fi

[ -n "$AWS_ACCESS_KEY_ID"     ] || abort "AWS_ACCESS_KEY_ID is not set"
[ -n "$AWS_SECRET_ACCESS_KEY" ] || abort "AWS_SECRET_ACCESS_KEY is not set"

for dir in ${CURR_DIR}/*/ ; do
    dir=`basename ${dir%*/}`
    echo $dir
    aws s3api put-object \
        --bucket "$AWS_BUCKET" \
        --key "yaml/${dir}/${version}"
    for f in ${CURR_DIR}/${dir}/*.yaml ; do
      fname=`basename $f`
      echo ${fname}
      aws s3api put-object \
        --bucket "$AWS_BUCKET" \
        --key "yaml/${dir}/${version}/$fname" \
        --body "$f" &&  echo "... yaml/${dir}/${version}/$fname pushed"
    done
done

exit 0
