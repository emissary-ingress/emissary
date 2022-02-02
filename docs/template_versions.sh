#!/bin/bash

set -e

usage() {
    echo "Usage: VERSION=v2.* template_versions.sh SOURCE_YAML_FILE DEST_YAML_FILE"
    exit 1
}

if [[ $# != 2 ]] || [[ ! -f "$1" ]]; then
    usage
fi
source_yaml="${1}"
dest_yaml="${2}"

if [[ ${VERSION:-} != v2.* ]]; then
    abort "VERSION must be set to a 'v2.*' string"
fi
version=${VERSION#v}

mkdir -p "$(dirname "$dest_yaml")"
rm -f "$dest_yaml"
sed \
    -e 's/\$version\$/'"${version}"'/g' \
    -e 's/\$quoteVersion\$/0.4.1/g' \
    <"$source_yaml" >"$dest_yaml"
