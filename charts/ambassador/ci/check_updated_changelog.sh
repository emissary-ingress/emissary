#!/bin/bash

set -e

CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir (maybe running in zsh?)";  exit 1; }
TOP_DIR=$CURR_DIR/..

# shellcheck source=common.sh
source "$CURR_DIR/common.sh"

echo ${TOP_DIR}
chart_version=$(get_chart_version ${TOP_DIR})

if ! grep "## v${chart_version}" ${TOP_DIR}/CHANGELOG.md > /dev/null 2>&1  ; then
    echo "Current chart version does not appear in the changelog."
    echo "Please run ambassador.git/charts/ambassador/ci/update_chart_changelog.sh and commit."
    exit 1
fi

echo "Changelog looks good!"
