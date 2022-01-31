#!/bin/env bash
set -e
set -o pipefail

>&2 echo "Scanning NPM package file $1"

DIR=$(dirname $1)

pushd "${DIR}" >/dev/null

>&2 cat package.json
>&2 echo "END package.json $1 ===================================================="


docker run --rm -i "js-deps-builder" sh -c 'tar xf - && ../scan.sh' \
  < <(tar cf - *) >&2
>&2 echo "END dependencies $1 ===================================================="

docker run --rm -i "js-deps-builder" sh -c 'tar xf - && ../scan.sh' \
   < <(tar cf - *) | ${JS_MKOPENSOURCE}

popd >/dev/null
