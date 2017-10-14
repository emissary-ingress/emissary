#!/bin/bash

export LC_ALL=C.UTF-8
export LANG=C.UTF-8

APPDIR=${APPDIR:-/application}

pids=()

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
        cat $LATEST
    else
        echo "No config generated."
    fi

    echo "AMBASSADOR: shutting down"
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
            # echo "AMBASSADOR: $name exited: $STATUS"
            # echo "AMBASSADOR: shutting down"
            diediedie "$name" "$STATUS"
        else
            tmp+=(${pids[i]})
        fi
    done

    pids=(${tmp[@]})
}

handle_int() {
    echo "Exiting due to Control-C"
}

set -o monitor
trap "handle_chld" CHLD
trap "handle_int" INT

/usr/bin/python3 "$APPDIR/kubewatch.py" sync /etc/ambassador-config /etc/envoy.json 

STATUS=$?

if [ $STATUS -ne 0 ]; then
    diediedie "kubewatch sync" "$STATUS"
fi

echo "AMBASSADOR: starting diagd"
/usr/bin/python3 "$APPDIR/diagd.py" --no-debugging /etc/ambassador-config &
pids+=("$!;diagd")

echo "AMBASSADOR: starting Envoy"
/usr/bin/python3 "$APPDIR/hot-restarter.py" "$APPDIR/start-envoy.sh" &
RESTARTER_PID="$!"
pids+=("${RESTARTER_PID};envoy")

/usr/bin/python3 "$APPDIR/kubewatch.py" watch /etc/ambassador-config /etc/envoy.json -p "${RESTARTER_PID}" &
pids+=("$!;kubewatch")

echo "AMBASSADOR: waiting"
wait
