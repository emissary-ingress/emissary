#!/bin/sh -e
cd $(dirname "$0")
. ../k8s-env.sh
BIN=../bin_$(go env GOHOSTOS)_$(go env GOHOSTARCH)
AMBASSADOR_KEYLOC=~/"Library/Application Support/ambassador"
test -d "$AMBASSADOR_KEYLOC" || mkdir "$AMBASSADOR_KEYLOC"
echo "$AMBASSADOR_LICENSE_KEY" > "$AMBASSADOR_KEYLOC/license-key"
export SHARED_SECRET_PATH="$AMBASSADOR_KEYLOC/shared-secret"
echo tbd > "$SHARED_SECRET_PATH"
export AMBASSADOR_URL=http://localhost:8877
export POLL_EVERY_SECS=20
python fake-ambassador.py & trap 'curl -v "$AMBASSADOR_URL/_shutdown"' INT EXIT
echo "======="
pwd
echo "======="
$BIN/dev-portal-server
