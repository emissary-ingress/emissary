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
    cat /etc/envoy.json

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

set -o monitor
trap "handle_chld" CHLD

echo "AMBASSADOR: checking /etc/envoy.json"
/usr/bin/python3 "$APPDIR/ambassador.py" config --check /etc/ambassador-config /etc/envoy.json

STATUS=$?

if [ $STATUS -ne 0 ]; then
    diediedie "ambassador" "$STATUS"
fi

echo "AMBASSADOR: starting diagd"
/usr/bin/python3 "$APPDIR/diagd.py" --no-debugging /etc/ambassador-config &
pids+=("$!;diagd")

echo "AMBASSADOR: starting Envoy"
/usr/local/bin/envoy -c /etc/envoy.json &
pids+=("$!;envoy")

echo "AMBASSADOR: waiting"
wait

