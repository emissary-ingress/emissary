#!/bin/bash

set -e

CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir";  exit 1; }
basedir=$1
shift
if [[ -z ${basedir} ]] || [[ ! -d ${basedir} ]]; then
    echo "must supply basedir as first argument"
    exit 1
fi
basedir=`realpath ${basedir}`/

[ -n "$AWS_ACCESS_KEY_ID"     ] || (echo "AWS_ACCESS_KEY_ID is not set" ; exit 1)
[ -n "$AWS_SECRET_ACCESS_KEY" ] || (echo "AWS_SECRET_ACCESS_KEY is not set" ; exit 1)

ver_yaml=${CURR_DIR}/yaml/versions.yml

version=$(grep version ${ver_yaml} | awk ' { print $2 }')
if [[ -n "${VERSION_OVERRIDE}" ]] ; then
    version=${VERSION_OVERRIDE}
fi

if [[ $version =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] ; then
    # if this is a stable version, working directory must be clean
    # otherwise this is an rc or test version and we don't care
    if [ -n "$(git status --porcelain)" ] ; then
        echo "working tree is dirty, aborting"
        exit 1
    fi
elif [[ "${BUMP_STABLE}" = "true" ]] ; then
    # if this isn't an X.Y.Z version, don't let allow bumping stable
    echo "Cannot bump stable unless this is an X.Y.Z tag"
    exit 1
fi

echo ${version} > stable.txt
if [ -z "$AWS_BUCKET" ] ; then
    AWS_BUCKET=datawire-static-files
fi

# make this something different than ambassador, emissary, or edge-stack
# so we don't conflict with the new hotness we're doing for 2.0
unversioned_base_s3_key=yaml/ambassador-docs/
base_s3_key=${unversioned_base_s3_key}${version}
aws s3api put-object \
    --bucket "$AWS_BUCKET" \
    --key ${base_s3_key}

echo "Pushing files to s3..."
for file in "$@"; do
    if [[ ! -f ${file} ]] ; then
        echo "${file} is not a file...."
        exit 1
    fi
    file=`realpath ${file}`
    s3_key=`echo ${file} | sed "s#${basedir}##"`
    s3_key="${base_s3_key}/${s3_key}"
    aws s3api put-object \
        --bucket "$AWS_BUCKET" \
        --key ${s3_key} \
        --body "$file" &&  echo "... ${s3_key} pushed"
done

if [[ "${BUMP_STABLE}" = "true" ]] ; then
    echo "Bumping stable version for yaml/${dir}"
    aws s3api put-object \
        --bucket "$AWS_BUCKET" \
        --key "${unversioned_base_s3_key}stable.txt" \
        --body stable.txt
fi

echo "Done pushing files to s3"
rm stable.txt
