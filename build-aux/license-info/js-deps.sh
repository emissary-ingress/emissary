#!/bin/env bash
set -e
set -o pipefail

>&2 echo "Scanning NPM package file $1"

DIR=$(dirname $1)

pushd "${DIR}" >/dev/null

>&2 cat package.json
>&2 echo "END package.json $1 ===================================================="


docker run --rm -i "js-deps-builder" sh -c 'tar xf - && ../scan.sh' \
  < <(tar cf - *) > tmp.tmp

cat tmp.tmp >&2
>&2 echo "END dependencies $1 ===================================================="

cat tmp.tmp | ${JS_MKOPENSOURCE}
rm -f tmp.tmp
popd >/dev/null
