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

if [[ ${CHART_VERSION:-} != v7.* ]]; then
    abort "CHART_VERSION must be set to a 'v7.*' string"
fi
chart_version=${CHART_VERSION#v}

new_changelog=${CHART_DIR}/CHANGELOG.new.md
rm ${new_changelog} > /dev/null 2>&1 || true
while IFS= read -r line ; do
    echo -e "${line}" >> ${new_changelog}
    if [[ "${line}" =~ "## Next Release" ]] ; then
        echo "" >> ${new_changelog}
        echo "(no changes yet)" >> ${new_changelog}
        echo "" >> ${new_changelog}
        echo "## v${chart_version}" >> ${new_changelog}
    fi

done < ${CHART_DIR}/CHANGELOG.md

mv ${new_changelog} ${CHART_DIR}/CHANGELOG.md

info "Done editing changelog for ${CHART_NAME}!"
