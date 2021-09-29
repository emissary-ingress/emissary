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
version=
if [[ -n "${VERSION_OVERRIDE}" ]] ; then
    version=${VERSION_OVERRIDE}
else
    version=$(grep version ${TOP_DIR}docs/yaml/versions.yml | awk '{ print $2 }')
fi
[ -n ${version} ] || abort "could not read version from docs/yaml/versions.yml"
log "Publishing manifest version ${version}"

if [[ $version =~ ^[0-9]+\.[0-9]+\.[0-9]+(-ea)?$ ]] ; then
    # if this is a stable version, working directory must be clean
    # otherwise this is an rc, ea or test version and we don't care
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

[ -n "$AWS_ACCESS_KEY_ID"     ] || abort "AWS_ACCESS_KEY_ID is not set"
[ -n "$AWS_SECRET_ACCESS_KEY" ] || abort "AWS_SECRET_ACCESS_KEY is not set"

echo ${version} > stable.txt
for dir in ${CURR_DIR}/*/ ; do
    dir=`basename ${dir%*/}`
    aws s3api put-object \
        --bucket "$AWS_S3_BUCKET" \
        --key "yaml/${dir}/${version}"
    for f in ${CURR_DIR}/${dir}/*.yaml ; do
      fname=`basename $f`
      echo ${fname}
      aws s3api put-object \
        --bucket "$AWS_S3_BUCKET" \
        --key "yaml/${dir}/${version}/$fname" \
        --body "$f" &&  echo "... yaml/${dir}/${version}/$fname pushed"
    done
    # bump the stable version for this directory
    if [[ "${BUMP_STABLE}" ]] ; then
        log "Bumping stable version for yaml/${dir}"
        aws s3api put-object \
            --bucket "$AWS_S3_BUCKET" \
            --key "yaml/${dir}/stable.txt" \
            --body stable.txt
    fi
done

rm stable.txt

exit 0
