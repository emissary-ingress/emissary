#!/bin/env bash
set -e
set -o pipefail

export DEPS_FILE=js-deps.json

function scan_dependencies() {
  set -e
  set -o pipefail

  echo >&2 "Scanning file $1"

  DIR=$(dirname $1)

  pushd "${DIR}" >/dev/null
  docker run --rm -i "js-deps-builder" sh -c 'tar xf - && ../scan.sh' \
    < <(tar cf - *) | ${JS_MKOPENSOURCE} >"${DEPS_FILE}"

  popd >/dev/null
}

export -f scan_dependencies

find . \( -path "./_cxx/envoy/*" \
  -o -path "./_generate.tmp/*" \
  \) -prune -o -name package.json -type f -exec bash -c 'scan_dependencies "{}"' \;
