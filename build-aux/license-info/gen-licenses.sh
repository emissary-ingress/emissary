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
sed 's/^---$//' "${PIP_SHOW}" | ${PY_MKOPENSOURCE} --output-type=json | jq -r '.licenseInfo | to_entries | .[] | "* [" + .key + "](" + .value + ")"' | sed -e 's/\[\([^]]*\)]()/\1/' >>"${LICENSES_TMP}"

# Analyze Node.Js dependencies
# TODO: Scan other folders with JS files but no package.json
find -name package.json -exec dirname {} \; | while IFS=$'\n' read packagedir; do
  pushd "${packagedir}" >/dev/null
  docker build -f "${DOCKERFILE}" --output "${TEMP}" .
  popd >/dev/null

  cat "${TEMP}/dependencies.json" | ${JS_MKOPENSOURCE} | jq -r '.licenseInfo | to_entries | .[] | "* [" + .key + "](" + .value + ")"' >>"${LICENSES_TMP}"
done

#Generate LICENSES.md
{
  echo -e "${APPLICATION} incorporates Free and Open Source software under the following licenses:\n"
  sort "${LICENSES_TMP}" | uniq | sed -e 's/\[\([^]]*\)]()/\1/'
} >"${DESTINATION}"
