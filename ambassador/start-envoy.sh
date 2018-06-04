#!/bin/sh

DRAIN_TIME=${AMBASSADOR_DRAIN_TIME:-5}
SHUTDOWN_TIME=${AMBASSADOR_SHUTDOWN_TIME:-10}

LATEST=$(ls -1v /etc/envoy*.json | tail -1)
exec /usr/local/bin/envoy -c ${LATEST} --restart-epoch $RESTART_EPOCH --service-cluster "${AMBASSADOR_ID:-ambassador}-${AMBASSADOR_NAMESPACE}" --drain-time-s "${DRAIN_TIME}" --parent-shutdown-time-s "${SHUTDOWN_TIME}"
