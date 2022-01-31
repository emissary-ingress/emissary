#!/bin/env bash
set -e
set -o pipefail

if [[ ! -f "${PIP_SHOW}" ]]; then
  echo >&2 "Could not find file ${PIP_SHOW}"
  exit 1
fi

cd "${BUILD_HOME}"

# Analyze Go dependencies
{
  ${GO_MKOPENSOURCE} --output-format=txt --package=mod --gotar=${GO_TAR}
  echo -e "\n"
} >"${DESTINATION}"

# Analyze Python dependencies
cat "${PIP_SHOW}" | ${PY_MKOPENSOURCE} | sed -e "s/Emissary-ingress/${APPLICATION}/" >>"${DESTINATION}"

# Analyze Node.Js dependencies
if [[ -f "${JS_DEPENDENCIES}" ]]; then
  function parse_js_dependencies() {
    jq -r '.dependencies[] | .name + "|" + .version + "|" + (.licenses | flatten | join(", "))' <"$1"
  }

  export -f parse_js_dependencies

  TMP_LICENSES="${BUILD_HOME}/_generate.tmp/licences"

  {
    echo -e "Name|Version|License(s)\n----|-------|----------"
    cat "${JS_DEPENDENCIES}"
  } >"${TMP_LICENSES}"

  {
    echo -e "\n\nThe ${APPLICATION} Node.Js code makes use of the following Free and Open Source\nlibraries:\n"

    awk 'BEGIN{OFS=FS="|"}
         NR==FNR {for (i=1; i<=NF; i++) max[i]=(length($i)>max[i]?length($i):max[i]); next}
                 {for (i=1; i<=NF; i++) printf "%s%-*s%s", i==1 ? "    " : "", i < NF? max[i]+2 : 1, $i, i==NF ? ORS : " "}
       ' "${TMP_LICENSES}" "${TMP_LICENSES}"
  } >>"${DESTINATION}"

  echo "============================================= Dependencies used to build final file"
  cat "${TMP_LICENSES}"
  echo "============================================= END"
fi