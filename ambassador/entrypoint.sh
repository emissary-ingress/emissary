#!/bin/bash

export LC_ALL=C.UTF-8
export LANG=C.UTF-8

CONFIG_DIR="/etc/ambassador-config"

if [ "$1" == "--demo" ]; then
    CONFIG_DIR="/etc/ambassador-demo-config"
fi

APPDIR=${APPDIR:-/application}

VERSION=$(python3 -c 'from ambassador.VERSION import Version; print(Version)')

pids=()

log () {
    now=$(date +"%Y-%m-%d %H:%M:%S")
    echo "$now AMBASSADOR: $@"
}

diediedie() {
    NAME=$1
    STATUS=$2

    if [ $STATUS -eq 0 ]; then
        log "$NAME claimed success, but exited \?\?\?\?"
    else
        log "$NAME exited with status $STATUS"
    fi

    log "Here's the envoy.json we were trying to run with:"
    LATEST="$(ls -v /etc/envoy*.json | tail -1)"
    if [ -e "$LATEST" ]; then
        cat $LATEST
    else
        log "No config generated."
    fi

    log "shutting down"
    exit 1
}

handle_chld() {
    local tmp=()

    for (( i=0; i<${#pids[@]}; ++i )); do
        split=(${pids[$i]//;/ })    # the space after the trailing / is critical!
        pid=${split[0]}
        name=${split[1]}

        if [ ! -d /proc/$pid ]; then
            wait $pid
            STATUS=$?
            diediedie "$name" "$STATUS"
        else
            tmp+=(${pids[i]})
        fi
    done

    pids=(${tmp[@]})
}

handle_int() {
    log "Exiting due to Control-C"
}

set -o monitor
trap "handle_chld" CHLD
trap "handle_int" INT

log "starting initial sync"

/usr/bin/python3 "$APPDIR/kubewatch.py" sync "$CONFIG_DIR" /etc/envoy.json 

STATUS=$?

if [ $STATUS -ne 0 ]; then
    diediedie "kubewatch sync" "$STATUS"
fi

if [ -z "$AMBASSADOR_NO_DIAGD" ]; then
    log "starting diagd"
    diagd --no-debugging "$CONFIG_DIR" &
    pids+=("$!;diagd")
fi

if [ -z "$AMBASSADOR_NO_ENVOY" ]; then
    log "starting Envoy"
    /usr/bin/python3 "$APPDIR/hot-restarter.py" "$APPDIR/start-envoy.sh" &
    RESTARTER_PID="$!"
    pids+=("${RESTARTER_PID};envoy")

    log "restarter PID $RESTARTER_PID"
    log "starting kubewatch"

    if [ -z "$AMBASSADOR_NO_KUBEWATCH" ]; then
        /usr/bin/python3 "$APPDIR/kubewatch.py" watch "$CONFIG_DIR" /etc/envoy.json -p "${RESTARTER_PID}" &
        pids+=("$!;kubewatch")
    fi
fi

log "ready for action"
wait
