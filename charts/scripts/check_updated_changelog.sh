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
CHART_DIR="${TOP_DIR}/${CHART_NAME}"

chart_version=$(get_chart_version ${CHART_DIR})

if ! grep "## v${chart_version}" ${CHART_DIR}/CHANGELOG.md > /dev/null 2>&1  ; then
    echo "Current chart version does not appear in the changelog."
    echo "Please run CHART_NAME=${CHART_NAME} ambassador.git/charts/scripts/update_chart_changelog.sh and commit."
    exit 1
fi

echo "Changelog looks good!"
