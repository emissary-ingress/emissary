#!/bin/sh

APPDIR=${APPDIR:-/application}
echo "$APPDIR"

/usr/bin/python3 "$APPDIR/envoy-restarter.py" /etc/envoy-restarter.pid "$APPDIR/envoy-wrapper.sh" &
/usr/bin/python3 "$APPDIR/ambassador.py" "$APPDIR/envoy-template.json" /etc/envoy.json /etc/envoy-restarter.pid

echo "Ambassador exiting" >&2

