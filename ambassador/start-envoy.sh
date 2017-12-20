#!/bin/bash
LATEST=$(ls -1v /etc/envoy*.json | tail -1)
exec /usr/local/bin/envoy -c ${LATEST} --restart-epoch $RESTART_EPOCH --drain-time-s 2 --parent-shutdown-time-s 3
