#!/bin/env bash
set -e

if [[ ! -f "${PIP_SHOW}" ]]; then
  echo >&2 "Could not find file ${PIP_SHOW}"
  exit 1
fi

TOOLS="${OSS_HOME}/tools"
DOCKERFILE=${OSS_HOME}/build-aux/license-info/docker/Dockerfile

LICENSES_TMP="${OSS_HOME}/_generate.tmp/LICENSES.md"
: >"${LICENSES_TMP}"

cd "${OSS_HOME}"

# Analyze Go dependencies
${GO_MKOPENSOURCE} --output-format=txt --package=mod --output-type=json --gotar=${GO_TAR} | jq -r '.licenseInfo | to_entries | .[] | "* [" + .key + "](" + .value + ")"' | sed -e 's/\[\([^]]*\)]()/\1/' >"${LICENSES_TMP}"

# Analyze Python dependencies
sed 's/^---$//' "${PIP_SHOW}" | ${PY_MKOPENSOURCE} --output-type=json | jq -r '.licenseInfo | to_entries | .[] | "* [" + .key + "](" + .value + ")"' | sed -e 's/\[\([^]]*\)]()/\1/' >>"${LICENSES_TMP}"

# Analyze Node.Js dependencies
function parse_js_dependencies() {
  jq -r '.licenseInfo | to_entries | .[] | "* [" + .key + "](" + .value + ")"' <"$1"
}

export -f parse_js_dependencies

find . -name "js-deps.json" -type f -exec bash -c 'parse_js_dependencies "{}"' \; | sed -e 's/\[\([^]]*\)]()/\1/' >>"${LICENSES_TMP}"

#Generate LICENSES.md
{
  echo -e "${APPLICATION} incorporates Free and Open Source software under the following licenses:\n"
  sort "${LICENSES_TMP}" | uniq | sed -e 's/\[\([^]]*\)]()/\1/'
} >"${DESTINATION}"
