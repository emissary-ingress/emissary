#!/bin/sh

set -ex

HERE=$(cd $(dirname $0); pwd)

cd "${HERE}/../ambassador"

python ambassador.py config test-config envoy-test.json

diff -u gold.json envoy-test.json
