#!/bin/bash

set -e

CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir (maybe running in zsh?)";  exit 1; }
TOP_DIR=$CURR_DIR/..

# shellcheck source=common.sh
source "$CURR_DIR/common.sh"

if [[ -z "${CHART_NAME}" ]] ; then
    abort "Need to specify the chart you wish to publish"
fi
chart_dir="${TOP_DIR}/${CHART_NAME}"

if [[ ! -d "${chart_dir}" ]] ; then
    abort "${chart_dir} is not a directory"
fi

#########################################################################################
if ! command -v gh 2> /dev/null ; then
    info "gh doesn't exist, install before continuing"
    exit 1
fi
if [[ ${CHART_VERSION:-} != v7.* ]]; then
    abort "CHART_VERSION must be set to a 'v7.*' string"
fi
thisversion=${CHART_VERSION#v}
chart_version=chart/v${thisversion}
git fetch

if ! git rev-parse ${chart_version} >/dev/null 2>&1 ; then
    info "${chart_version} isnt a git tag, aborting"
    exit 1
fi

if [[ $thisversion =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] ; then
    create_chart_release $thisversion $chart_dir
else
    info "${thisversion} doesnt look like a GA version, not creating release"
fi


exit 0
