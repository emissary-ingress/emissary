#!/bin/bash

set -e

usage() {
    echo "USAGE: template_versions.sh [SOURCE YAML] [DEST YAML]"
    exit 1
}

if [[ -z "${1}" ]] || [[ -z "${2}" ]] ; then
    usage
fi
if [[ ! -f "${1}" ]] ; then
    usage
fi
CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir";  exit 1; }

source_yaml="${1}"
dest_yaml="${2}"
mkdir -p `dirname ${dest_yaml}`
ver_yaml=${CURR_DIR}/yaml/versions.yml
cp ${source_yaml} ${dest_yaml}

while read -r line; do
    if [[ -z "${line}" ]] ; then
        continue
    fi
    value=`echo ${line} | awk '{split($0,a,":"); print a[2]}'`
    key=`echo ${line} | awk '{split($0,a,":"); print a[1]}'`
    key="\\\$${key}\\\$"
    value=`echo ${value} | sed 's/ *$//g'`
    sed -i.bak "s/${key}/${value}/g" ${dest_yaml}
    rm ${dest_yaml}.bak
done < ${ver_yaml}
