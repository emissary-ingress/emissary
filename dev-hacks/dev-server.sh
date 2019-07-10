#!/bin/sh -e
cd $(dirname "$0")
. ../k8s-env.sh
AMBASSADOR_KEYLOC=~/"Library/Application Support/ambassador"
test -d "$AMBASSADOR_KEYLOC" || mkdir "$AMBASSADOR_KEYLOC"
echo "$AMBASSADOR_LICENSE_KEY" > "$AMBASSADOR_KEYLOC/license-key"
export SHARED_SECRET_PATH="$AMBASSADOR_KEYLOC/shared-secret"
echo tbd > "$SHARED_SECRET_PATH"
export AMBASSADOR_URL=http://localhost:8877
export POLL_EVERY_SECS=5
python fake-ambassador.py & trap 'curl -v "$AMBASSADOR_URL/_shutdown"' INT EXIT
../bin_darwin_amd64/dev-portal-server
