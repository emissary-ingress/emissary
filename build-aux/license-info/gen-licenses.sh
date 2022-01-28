#!/bin/env bash
set -ex

if [[ ! -f "${PIP_SHOW}" ]]; then
  >&2 echo "Could not find file ${PIP_SHOW}"
  exit 1
fi

TOOLS="${OSS_HOME}/tools"
DOCKERFILE=${OSS_HOME}/build-aux/license-info/docker/Dockerfile

TEMP="${OSS_HOME}/_generate.tmp/licences"
mkdir -p "${TEMP}"
LICENSES_TMP=${TEMP}/LICENSES.md
: >"${LICENSES_TMP}"

cd "${OSS_HOME}"

# Analyze Go dependencies
${GO_MKOPENSOURCE} --output-format=txt --package=mod --output-type=json --gotar=${GO_TAR} | jq -r '.licenseInfo | to_entries | .[] | "* [" + .key + "](" + .value + ")"' | sed -e 's/\[\([^]]*\)]()/\1/' >"${LICENSES_TMP}"

# Analyze Python dependencies
{
  sed 's/^---$//' "${PIP_SHOW}"
  echo
} | ${PY_MKOPENSOURCE} --output-type=json | jq -r '.licenseInfo | to_entries | .[] | "* [" + .key + "](" + .value + ")"' | sed -e 's/\[\([^]]*\)]()/\1/' >>"${LICENSES_TMP}"

#Generate LICENSES.md
{
  echo -e "${APPLICATION} incorporates Free and Open Source software under the following licenses:\n"
  sort "${LICENSES_TMP}" | uniq | sed -e 's/\[\([^]]*\)]()/\1/'
} >"${DESTINATION}"
