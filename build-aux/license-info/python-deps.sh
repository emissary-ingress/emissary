#!/bin/env bash
set -e

echo >&2 "Scanning python requirements $1"

DIR=$(dirname $1)
pushd "${DIR}" >/dev/null

(
  docker run --rm -i "python-deps-builder" sh -c 'tar xf - && pip3 --disable-pip-version-check install -r requirements.txt >/dev/null && pip3 --disable-pip-version-check freeze --exclude-editable  | cut -d= -f1 | xargs pip show' \
    < <(tar cf - requirements.txt)

  echo '---'
) | sed 's/^---$//'

popd >/dev/null
