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

cd "${OSS_HOME}"

# Analyze Go dependencies
{
  ${GO_MKOPENSOURCE} --output-format=txt --package=mod --gotar=${GO_TAR}
  echo -e "\n"
} >"${DESTINATION}"


# Analyze Python dependencies
sed 's/^---$//' "${PIP_SHOW}" | ${PY_MKOPENSOURCE} >>"${DESTINATION}"
