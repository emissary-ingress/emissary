#!/bin/bash

export LC_ALL=C.UTF-8
export LANG=C.UTF-8

APPDIR=${APPDIR:-/demo}
echo "DOCKER-DEMO: args $@"

pids=()

handle_chld() {
    local tmp=()

    for (( i=0; i<${#pids[@]}; ++i )); do
        split=(${pids[$i]//;/ })    # the space after the trailing / is critical!
        pid=${split[0]}
        name=${split[1]}

        if [ ! -d /proc/$pid ]; then
            wait $pid
            echo "DOCKER-DEMO: $name exited: $?"
            echo "DOCKER-DEMO: shutting down"
            exit 1
        else
            tmp+=(${pids[i]})
        fi
    done

    pids=(${tmp[@]})
}

set -o monitor
trap "handle_chld" CHLD

ROOT=$$

if python3 check-services.py; then
    echo "DOCKER-DEMO: starting QoTM"
    /usr/bin/python3 "$APPDIR/qotm.py" &
    pids+=("$!;qotm")

    echo "DOCKER-DEMO: starting Extauth"
    /usr/bin/python3 "$APPDIR/simple-auth-server.py" &
    pids+=("$!;extauth")

    echo "DOCKER-DEMO: starting probes"
    /bin/sh "$APPDIR/probes.sh" &
    pids+=("$!;probes")
fi

echo "DOCKER-DEMO: starting Ambassador"
( cd /application ; /bin/bash entrypoint.sh ) &
pids+=("$!;ambassador")

echo "DOCKER-DEMO: waiting"
wait

