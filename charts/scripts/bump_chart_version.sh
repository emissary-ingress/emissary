#!/bin/bash

set -e

usage() {
    echo "USAGE:"
    echo "bump_version.sh (major|minor|revision) [CHART_YAML]"
    exit 1
}

bump_type=$1
shift
chart_yaml=$1
shift
version_to_bump=`grep 'version:' $chart_yaml | sed -E 's/version: ([0-9]+\.[0-9]+\.[0-9]+).*/\1/g'`

major=
minor=
patch=

if [[ "${version_to_bump}" =~ ^([0-9]+)\.([0-9]+)\.([0-9]+)$ ]] ; then
    major=${BASH_REMATCH[1]}
    minor=${BASH_REMATCH[2]}
    patch=${BASH_REMATCH[3]}
else
    echo "${version_to_bump} is not \d+.\d+.\d+"
    exit 1
fi

new_ver=

case "$bump_type" in
    major) new_ver="$((major + 1)).0.0";;
    minor) new_ver="${major}.$((minor + 1)).0";;
    patch) new_ver="${major}.${minor}.$((patch + 1))";;
    *) usage ;;
esac

if [[ ! -f "${chart_yaml}" ]] ; then
    echo "${chart_yaml} is not a file"
    usage
fi
echo $new_ver
sed -i.bak -E "s/version: [0-9]+\.[0-9]+\.[0-9]+.*/version: ${new_ver}/g" ${chart_yaml}
rm ${chart_yaml}.bak
