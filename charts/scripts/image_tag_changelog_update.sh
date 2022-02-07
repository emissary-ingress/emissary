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
if [[ -z "${IMAGE_TAG}" ]] ; then
    abort "Need to specify the image tag being updated"
fi

CHART_DIR="${TOP_DIR}/${CHART_NAME}"

if [[ ${CHART_VERSION:-} != v7.* ]]; then
    abort "CHART_VERSION must be set to a 'v7.*' string"
fi
chart_version=${CHART_VERSION#v}

new_changelog=${CHART_DIR}/CHANGELOG.new.md
ambassador_changelog_link="https://github.com/emissary-ingress/emissary/blob/master/CHANGELOG.md"
rm ${new_changelog} > /dev/null 2>&1 || true
buffering=
while IFS= read -r line ; do
    if [[ "${line}" =~ "## Next Release" ]] ; then
        buffering=true
    elif [[ "${buffering}" ]] && [[ "${line}" != "" ]]; then
        buffering=
        echo "- Update Ambassador chart image to version v${IMAGE_TAG}: [CHANGELOG](${ambassador_changelog_link})" >> ${new_changelog}
        if [[ "${line}" =~ (no changes yet) ]] ; then
            continue
        fi
    fi
    echo -e "${line}" >> ${new_changelog}

done < ${CHART_DIR}/CHANGELOG.md

mv ${new_changelog} ${CHART_DIR}/CHANGELOG.md

info "Done editing changelog for ${CHART_NAME}!"
