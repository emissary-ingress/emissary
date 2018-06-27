#!/bin/sh

# Copyright 2018 Datawire. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License

export LC_ALL=C.UTF-8
export LANG=C.UTF-8

AMBASSADOR_ROOT="/ambassador"
CONFIG_DIR="$AMBASSADOR_ROOT/ambassador-config"
ENVOY_CONFIG_FILE="$AMBASSADOR_ROOT/envoy.json"

export PYTHON_EGG_CACHE=$(AMBASSADOR_ROOT)

if [ "$1" == "--demo" ]; then
    CONFIG_DIR="$AMBASSADOR_ROOT/ambassador-demo-config"
fi

DELAY=${AMBASSADOR_RESTART_TIME:-15}

APPDIR=${APPDIR:-"$AMBASSADOR_ROOT"}

# If we don't set PYTHON_EGG_CACHE explicitly, /.cache is set by default, which fails when running as a non-privileged
# user
export PYTHON_EGG_CACHE=${APPDIR/.cache}

export PYTHONUNBUFFERED=true

pids=""

ambassador_exit() {
    RC=${1:-0}

    if [ -n "$AMBASSADOR_EXIT_DELAY" ]; then
        echo "AMBASSADOR: sleeping for debug"
        sleep $AMBASSADOR_EXIT_DELAY
    fi

    echo "AMBASSADOR: shutting down ($RC)"
    exit $RC
}

diediedie() {
    NAME=$1
    STATUS=$2

    if [ $STATUS -eq 0 ]; then
        echo "AMBASSADOR: $NAME claimed success, but exited \?\?\?\?"
    else
        echo "AMBASSADOR: $NAME exited with status $STATUS"
    fi

    echo "Here's the envoy.json we were trying to run with:"
    LATEST="$(ls -v $AMBASSADOR_ROOT/envoy*.json | tail -1)"
    if [ -e "$LATEST" ]; then
        cat "$LATEST"
    else
        echo "No config generated."
    fi
    
    ambassador_exit 1
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

# set -o monitor
trap "handle_chld" CHLD
trap "handle_int" INT

/usr/bin/python3 "$APPDIR/kubewatch.py" sync "$CONFIG_DIR" "$ENVOY_CONFIG_FILE"

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

/usr/bin/python3 "$APPDIR/kubewatch.py" watch "$CONFIG_DIR" "$ENVOY_CONFIG_FILE" -p "${RESTARTER_PID}" --delay "${DELAY}" &
pids="${pids:+${pids} }$!:kubewatch"

echo "AMBASSADOR: waiting"
echo "PIDS: $pids"
wait

ambassador_exit 0
