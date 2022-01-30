#!/bin/env bash
set -e

if [[ ! -f "${PIP_SHOW}" ]]; then
  echo >&2 "Could not find pip dependency file ${PIP_SHOW}"
  exit 1
fi

cd "${BUILD_HOME}"

#Generate LICENSES.md
{
  echo -e "${APPLICATION} incorporates Free and Open Source software under the following licenses:\n"

  {
    # Analyze Go dependencies
    ${GO_MKOPENSOURCE} --output-format=txt --package=mod --output-type=json --gotar=${GO_TAR} | jq -r '.licenseInfo | to_entries | .[] | "* [" + .key + "](" + .value + ")"' | sed -e 's/\[\([^]]*\)]()/\1/'

    # Analyze Python dependencies
    cat "${PIP_SHOW}" | ${PY_MKOPENSOURCE} --output-type=json | jq -r '.licenseInfo | to_entries | .[] | "* [" + .key + "](" + .value + ")"' | sed -e 's/\[\([^]]*\)]()/\1/'

    if [[ -e "${JS_LICENSES}" ]]; then
      # Analyze Node.Js dependencies
      cat "${JS_LICENSES}"
    fi
  } | sort | uniq

} >"${DESTINATION}"
