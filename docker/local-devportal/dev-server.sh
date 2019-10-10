#!/bin/sh -e
cd $(dirname "$0")
export AMBASSADOR_INTERNAL_URL=http://localhost:8877
export POLL_EVERY_SECS=200
export CODE_CONTENT_URL="/content"
export AMBASSADOR_LICENSE_FILE="$CODE_CONTENT_URL/.devportal.license"

python3 /usr/local/ambassador/fake-ambassador.py &
sleep 1
curl -qs --retry 4 --retry-connrefused $AMBASSADOR_INTERNAL_URL
echo "======="
/usr/local/ambassador/local-devportal
