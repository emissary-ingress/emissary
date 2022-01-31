#!/bin/env bash
set -e
set -o pipefail

echo >&2 "Scanning NPM package file $1"

DIR=$(dirname $1)

pushd "${DIR}" >/dev/null
docker run --rm -i "js-deps-builder" sh -c 'tar xf - && ../scan.sh' \
  < <(tar cf - *) | ${JS_MKOPENSOURCE}

popd >/dev/null
