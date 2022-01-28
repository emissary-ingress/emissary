#!/bin/env bash
set -e

function scan_dependencies() {
  >&2 echo "Scannning file $1"

  DIR=$(dirname $1)
  pushd "${DIR}" >/dev/null

  docker run --rm -i "python-deps-builder" sh -c 'tar xf - && pip3 --disable-pip-version-check install -r requirements.txt >/dev/null && pip3 --disable-pip-version-check freeze --exclude-editable  | cut -d= -f1 | xargs pip show' \
    < <(tar cf - requirements.txt) \

  echo '---'
  popd >/dev/null
}

export -f scan_dependencies

find . \( -path "./_cxx/envoy/*" \
  -o -path "./docker/test-auth/*" \
  -o -path "./docker/test-shadow/*" \
  -o -path "./docker/test-stats/*" \
  -o -path "./_generate.tmp/*" \
  \) -prune -o -name requirements.txt -type f -exec bash -c 'scan_dependencies "{}"' \;
