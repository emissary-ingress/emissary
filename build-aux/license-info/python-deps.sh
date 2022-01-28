#!/bin/env bash
set -e

DIR=$(dirname $1)
pushd "${DIR}" >/dev/null

docker run --rm -i "python-deps-builder" sh -c 'tar xf - && pip3 --disable-pip-version-check install -r requirements.txt >/dev/null && pip3 --disable-pip-version-check freeze --exclude-editable  | cut -d= -f1 | xargs pip show' \
       < <(tar cf - requirements.txt) \
       >pip-deps.txt
echo '---' >>pip-deps.txt
popd >/dev/null
