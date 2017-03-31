#!/bin/sh

/usr/local/bin/envoy -c /etc/envoy.json --restart-epoch $RESTART_EPOCH
