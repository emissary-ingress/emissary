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

export PYTHON_EGG_CACHE=${AMBASSADOR_ROOT}

if [ "$1" == "--demo" ]; then
    CONFIG_DIR="$AMBASSADOR_ROOT/ambassador-demo-config"
fi

DELAY=${AMBASSADOR_RESTART_TIME:-1}

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

wait_for_ready() {
    host=$1
    is_ready=1
    sleep_for_seconds=4
    while true; do
        sleep ${sleep_for_seconds}
        if getent hosts statsd-sink; then
            echo "$host exists"
            is_ready=0
            break
        else
            echo "$host is not reachable, trying again in ${sleep_for_seconds} seconds ..."
        fi
    done
    return ${is_ready}
}

handle_statsd() {
    STATSD_HOST=$1
    echo "Waiting till $STATSD_HOST is reachable"

    if wait_for_ready ${STATSD_HOST}; then
        echo "$STATSD_HOST is reachable, starting socat ..."
        socat -d UDP-RECVFROM:8125,fork UDP-SENDTO:${STATSD_HOST}:8125
    fi
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

echo "AMBASSADOR: starting ads"
./ambex /ambassador/envoy &
AMBEX_PID="$!"
pids="${pids:+${pids} }${AMBEX_PID}:ambex"

echo "AMBASSADOR: starting Envoy"
envoy -c bootstrap-ads.yaml &
pids="${pids:+${pids} }$!:envoy"

/usr/bin/python3 "$APPDIR/kubewatch.py" watch "$CONFIG_DIR" "$ENVOY_CONFIG_FILE" -p "${AMBEX_PID}" --delay "${DELAY}" &
pids="${pids:+${pids} }$!:kubewatch"

if [ "$(echo ${STATSD_ENABLED} | tr "[:upper:]" "[:lower:]")" = "true" ]; then
    echo "STATSD_ENABLED is set to true"
    # Fallback to statsd-sink if a host isn't provided
    STATSD_HOST="${STATSD_HOST:-statsd-sink}"
    handle_statsd ${STATSD_HOST} &
else
    echo "STATSD_ENABLED is not set to true, no stats will be exposed"
fi

echo "AMBASSADOR: waiting"
echo "PIDS: $pids"
wait

ambassador_exit 0
