#!/bin/sh

/usr/local/bin/envoy -c /application/envoy.json --restart-epoch $RESTART_EPOCH
