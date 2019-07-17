#!/bin/sh -e
cd $(dirname "$0")
if [ ! -d .venv ]; then
	virtualenv --python=python3 .venv
        .venv/bin/pip install bottle WSGIProxy2
fi
. ../k8s-env.sh
BIN=../bin_$(go env GOHOSTOS)_$(go env GOHOSTARCH)
AMBASSADOR_KEYLOC=$(pwd)
export AMBASSADOR_LICENSE_FILE="$AMBASSADOR_KEYLOC/.license-key"
echo "$AMBASSADOR_LICENSE_KEY" > "$AMBASSADOR_LICENSE_FILE"
export SHARED_SECRET_PATH="$AMBASSADOR_KEYLOC/.shared-secret"
echo tbd > "$SHARED_SECRET_PATH"
export AMBASSADOR_URL=http://localhost:8877
export POLL_EVERY_SECS=20
.venv/bin/python fake-ambassador.py & trap 'curl "$AMBASSADOR_URL/_shutdown"; kill %1' EXIT
echo "======="
pwd
echo "======="
$BIN/dev-portal-server
