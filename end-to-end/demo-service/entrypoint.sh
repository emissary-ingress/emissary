#!/bin/bash

export LC_ALL=C.UTF-8
export LANG=C.UTF-8

APPDIR=${APPDIR:-/application}

env | grep V
echo "DEMO: args $@"

pids=()

handle_chld() {
    local tmp=()

    for (( i=0; i<${#pids[@]}; ++i )); do
        split=(${pids[$i]//;/ })    # the space after the trailing / is critical!
        pid=${split[0]}
        name=${split[1]}

        if [ ! -d /proc/$pid ]; then
            wait $pid
            echo "DEMO: $name exited: $?"
            echo "DEMO: shutting down"
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

echo "DEMO: starting demo service"
/usr/bin/python3 "$APPDIR/demo.py" "$@" &
pids+=("$!;demo")

echo "DEMO: waiting"
wait

