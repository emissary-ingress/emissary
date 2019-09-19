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
export AMBASSADOR_INTERNAL_URL=http://localhost:8877
export POLL_EVERY_SECS=200

content_url_file=.content-url-file
if [ -f $content_url_file ]; then
    . $content_url_file
else
    echo "You need to setup $content_url_file in $PWD"
    echo ""
    echo '  # echo export CODE_CONTENT_URL="https://github.com/datawire/devportal-content.git"'" > $content_url_file"
    echo ""
    false
fi

.venv/bin/python fake-ambassador.py & trap 'curl "$AMBASSADOR_INTERNAL_URL/_shutdown"; kill %1' EXIT

if [ "$(type -p socat)" != "" ]; then
    echo | socat - TCP:localhost:8877,retry=5,interval=1 >/dev/null 2>&1
else
    sleep 2
fi
echo "======="
pwd
echo "======="
$BIN/local-devportal
