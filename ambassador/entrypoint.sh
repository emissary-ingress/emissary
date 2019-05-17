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

if [ -z "$AMBASSADOR_NAMESPACE" ]; then
    AMBASSADOR_NAMESPACE=default
fi

AMBASSADOR_ROOT="/ambassador"
AMBASSADOR_CONFIG_BASE_DIR="${AMBASSADOR_CONFIG_BASE_DIR:-$AMBASSADOR_ROOT}"
CONFIG_DIR="${AMBASSADOR_CONFIG_BASE_DIR}/ambassador-config"
SNAPSHOT_DIR="${AMBASSADOR_CONFIG_BASE_DIR}/snapshots"

ENVOY_DIR="${AMBASSADOR_CONFIG_BASE_DIR}/envoy"
export ENVOY_DIR

ENVOY_CONFIG_FILE="${ENVOY_DIR}/envoy.json"
TEMP_ENVOY_CONFIG_FILE="/tmp/envoy.json"

# The bootstrap file really is in the config base dir, not the Envoy dir.
ENVOY_BOOTSTRAP_FILE="${AMBASSADOR_CONFIG_BASE_DIR}/bootstrap-ads.json"
export ENVOY_BOOTSTRAP_FILE

# Set AMBASSADOR_DEBUG to things separated by spaces to enable debugging.
check_debug () {
    word="$1"
    args="$2"

    # I'm not sure if ${x:---debug} works, but it's too weird to read anyway.
    if [ -z "$args" ]; then
        args="--debug"
    fi

    if [ $(echo "$AMBASSADOR_DEBUG" | grep -c "$word" || :) -gt 0 ]; then
        echo "$args"
    else
        echo ""
    fi
}

DIAGD_DEBUG=$(check_debug "diagd")
KUBEWATCH_DEBUG=$(check_debug "kubewatch")

ENVOY_DEBUG=$(check_debug "envoy" "-l debug")
export ENVOY_DEBUG

DIAGD_CONFIGDIR=

echo "AMBASSADOR STARTING with environment:"
echo "===="
env | grep AMBASSADOR | sort
echo "===="

if [ "$1" == "--demo" ]; then
    # This is _not_ meant to be overridden by AMBASSADOR_CONFIG_BASE_DIR.
    # It's baked into a specific location during the build process.
    CONFIG_DIR="$AMBASSADOR_ROOT/ambassador-demo-config"

    PORT=5050 python3 demo-services/auth.py &
    python3 demo-services/qotm.py &
fi

# Do we have config on the filesystem?

if [ $(find "${CONFIG_DIR}" -type f 2>/dev/null | wc -l) -gt 0 ]; then
    echo "AMBASSADOR: using $CONFIG_DIR for configuration"

    # XXX This won't work if CONFIG_DIR contains a space. Sigh.
    DIAGD_CONFIGDIR="--config-path ${CONFIG_DIR}"

    # Don't watch for Kubernetes changes.
    if [ -z "${AMBASSADOR_FORCE_KUBEWATCH}" ]; then
        echo "AMBASSADOR: not watching for Kubernetes config"
        AMBASSADOR_NO_KUBEWATCH=no_kubewatch
    fi
fi

mkdir -p "${SNAPSHOT_DIR}"
mkdir -p "${ENVOY_DIR}"

APPDIR=${APPDIR:-"$AMBASSADOR_ROOT"}

# If we don't set PYTHON_EGG_CACHE explicitly, /.cache is set by default, which fails when running as a non-privileged
# user
export PYTHON_EGG_CACHE="${PYTHON_EGG_CACHE:-$APPDIR}/.cache"

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

# set -o monitor
trap "handle_chld" CHLD
trap "handle_int" INT

#KUBEWATCH_DEBUG="--debug"

# Start using ancient kubewatch to get our cluster ID.
# XXX Ditch this, really.
cluster_id=$(/usr/bin/python3 "$APPDIR/kubewatch.py" --debug) #$KUBEWATCH_DEBUG)

STATUS=$?

if [ $STATUS -ne 0 ]; then
    diediedie "kubewatch cluster-id" "$STATUS"
fi

# Set Ambassador's cluster ID here. We can do this unconditionally because if AMBASSADOR_CLUSTER_ID was set
# before, kubewatch sync will use it.
AMBASSADOR_CLUSTER_ID="${cluster_id}"
export AMBASSADOR_CLUSTER_ID
echo "AMBASSADOR: using cluster ID $AMBASSADOR_CLUSTER_ID"

# Empty Envoy directory, hence no config via ADS yet.
mkdir -p "${ENVOY_DIR}"

if [ -z "${DIAGD_ONLY}" ]; then
    echo "AMBASSADOR: starting ads"
    ambex -ads 8003 "${ENVOY_DIR}" &
    AMBEX_PID="$!"
    pids="${pids:+${pids} }${AMBEX_PID}:ambex"
else
    DIAGD_EXTRA="--no-checks --no-envoy"
fi

# We can't start Envoy until the initial config happens, which means that diagd has to start it.

echo "AMBASSADOR: starting diagd"

diagd "${SNAPSHOT_DIR}" "${ENVOY_BOOTSTRAP_FILE}" "${ENVOY_CONFIG_FILE}" $DIAGD_DEBUG $DIAGD_CONFIGDIR \
      --kick "sh /ambassador/kick_ads.sh $AMBEX_PID" --notices "${AMBASSADOR_CONFIG_BASE_DIR}/notices.json" $DIAGD_EXTRA &
pids="${pids:+${pids} }$!:diagd"

# Wait for diagd to start
tries_left=10
delay=1
while [ $tries_left -gt 0 ]; do
    echo "AMBASSADOR: pinging diagd ($tries_left)..."

    status=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8877/_internal/v0/ping)

    if [ "$status" = "200" ]; then
        break
    fi

    tries_left=$(( $tries_left - 1 ))
    sleep $delay
    delay=$(( $delay * 2 ))
    if [ $delay -gt 10 ]; then delay=5; fi
done

if [ $tries_left -le 0 ]; then
    echo "AMBASSADOR: giving up on diagd and hoping for the best..."
else
    echo "AMBASSADOR: diagd running"
fi

if [ -z "${AMBASSADOR_NO_KUBEWATCH}" ]; then
#    KUBEWATCH_SYNC_CMD="python3 /ambassador/post_update.py"
    KUBEWATCH_SYNC_CMD="sh /ambassador/post_watt.sh"
    WATCH_HOOK="/ambassador/watch_hook.py"

    KUBEWATCH_NAMESPACE_ARG=""

    if [ -n "$AMBASSADOR_SINGLE_NAMESPACE" ]; then
        KUBEWATCH_NAMESPACE_ARG="--namespace $AMBASSADOR_NAMESPACE"
    fi

    KUBEWATCH_SYNC_KINDS="-s service -s AuthService -s Mapping -s Module -s RateLimitService -s TCPMapping -s TLSContext -s TracingService"

#    if [ -n "$AMBASSADOR_NO_SECRETS" ]; then
#        KUBEWATCH_SYNC_KINDS="-s service"
#    fi

    set -x
    /ambassador/watt ${KUBEWATCH_NAMESPACE_ARG} --port 8002 --notify "$KUBEWATCH_SYNC_CMD" $KUBEWATCH_SYNC_KINDS --watch "$WATCH_HOOK" &
    set +x
    pids="${pids:+${pids} }$!:watt"
fi

echo "AMBASSADOR: waiting"
echo "PIDS: $pids"
wait

ambassador_exit 0
