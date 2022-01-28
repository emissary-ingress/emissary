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
(
  {
    sed 's/^---$//' "${PIP_SHOW}"
    echo
  } | ${PY_MKOPENSOURCE}
  echo -e "\n"
) >>"${DESTINATION}"


# Analyze Node.Js dependencies
# TODO: Scan other folders with JS files but no package.json
echo -e "The ${APPLICATION} Node.Js code makes use of the following Free and Open Source
libraries:\n" >>"${DESTINATION}"

(
  echo -e "Name|Version|License(s)
----|-------|----------"

  {
    find . -name package.json -exec dirname {} \; | while IFS=$'\n' read packagedir; do
    pushd "${packagedir}" >/dev/null
    docker build -f "${DOCKERFILE}" --output "${TEMP}" . >&2
    popd >/dev/null

    cat "${TEMP}/dependencies.json" | ${JS_MKOPENSOURCE} | jq -r '.dependencies[] | .name + "|" + .version + "|" + (.licenses | flatten | join(", "))'
  done
  } | sort | uniq | sed -e 's/\[\([^]]*\)]()/\1/'
) > "${TEMP}/output"

awk 'BEGIN{OFS=FS="|"}
       NR==FNR {for (i=1; i<=NF; i++) max[i]=(length($i)>max[i]?length($i):max[i]); next}
               {for (i=1; i<=NF; i++) printf "%s%-*s%s", i==1 ? "    " : "", i < NF? max[i]+2 : 1, $i, i==NF ? ORS : " "}
     ' "${TEMP}/output" "${TEMP}/output" >> "${DESTINATION}"
