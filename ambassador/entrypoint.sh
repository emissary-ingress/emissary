#!/bin/sh

export LC_ALL=C.UTF-8
export LANG=C.UTF-8

CONFIG_DIR="/etc/ambassador-config"

if [ "$1" == "--demo" ]; then
    CONFIG_DIR="/etc/ambassador-demo-config"
fi

APPDIR=${APPDIR:-/application}

pids=""

diediedie() {
    NAME=$1
    STATUS=$2

    if [ $STATUS -eq 0 ]; then
        echo "AMBASSADOR: $NAME claimed success, but exited \?\?\?\?"
    else
        echo "AMBASSADOR: $NAME exited with status $STATUS"
    fi

    echo "Here's the envoy.json we were trying to run with:"
    LATEST="$(ls -v /etc/envoy*.json | tail -1)"
    if [ -e "$LATEST" ]; then
        cat "$LATEST"
    else
        echo "No config generated."
    fi

    echo "AMBASSADOR: shutting down"
    exit 1
}

handle_chld() {
    trap - CHLD
    local tmp
    for entry in $pids; do
        local pid="${entry%:*}"
        local name="${entry#*:}"
        if [ ! -d "/proc/${pid}" ]; then
            wait "${pid}"
            STATUS=$?
            # echo "AMBASSADOR: $name exited: $STATUS"
            # echo "AMBASSADOR: shutting down"
            diediedie "${name}" "$STATUS"
        else
            tmp="${tmp:+${tmp} }${entry}"
        fi
    done

    pids="$tmp"
    trap "handle_chld" CHLD
}

handle_int() {
    echo "Exiting due to Control-C"
}

set -o monitor
trap "handle_chld" CHLD
trap "handle_int" INT

/usr/bin/python3 "$APPDIR/kubewatch.py" sync "$CONFIG_DIR" /etc/envoy.json

STATUS=$?

if [ $STATUS -ne 0 ]; then
    diediedie "kubewatch sync" "$STATUS"
fi

echo "AMBASSADOR: starting diagd"
diagd --no-debugging "$CONFIG_DIR" &
pids="${pids:+${pids} }$!:diagd"

echo "AMBASSADOR: starting Envoy"
/usr/bin/python3 "$APPDIR/hot-restarter.py" "$APPDIR/start-envoy.sh" &
RESTARTER_PID="$!"
pids="${pids:+${pids} }${RESTARTER_PID}:envoy"

/usr/bin/python3 "$APPDIR/kubewatch.py" watch "$CONFIG_DIR" /etc/envoy.json -p "${RESTARTER_PID}" &
pids="${pids:+${pids} }$!:kubewatch"

echo "AMBASSADOR: waiting"
echo "PIDS: $pids"
wait
