#!/bin/bash

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

in_array() {
    local needle straw haystack
    needle="$1"
    haystack=("${@:2}")
    for straw in "${haystack[@]}"; do
        if [[ "$straw" == "$needle" ]]; then
            return 0
        fi
    done
    return 1
}

################################################################################
# CONFIG PARSING                                                               #
################################################################################

ambassador_root="/ambassador"

export LC_ALL=C.UTF-8
export LANG=C.UTF-8

export AMBASSADOR_NAMESPACE="${AMBASSADOR_NAMESPACE:-default}"
export AMBASSADOR_CONFIG_BASE_DIR="${AMBASSADOR_CONFIG_BASE_DIR:-$ambassador_root}"
export ENVOY_DIR="${AMBASSADOR_CONFIG_BASE_DIR}/envoy"
export ENVOY_BOOTSTRAP_FILE="${AMBASSADOR_CONFIG_BASE_DIR}/bootstrap-ads.json"

export APPDIR="${APPDIR:-$ambassador_root}"

# If we don't set PYTHON_EGG_CACHE explicitly, /.cache is set by
# default, which fails when running as a non-privileged user
export PYTHON_EGG_CACHE="${PYTHON_EGG_CACHE:-$AMBASSADOR_CONFIG_BASE_DIR}/.cache"
export PYTHONUNBUFFERED=true

ENTRYPOINT_DEBUG=

if [[ "$1" == "--dev-magic" ]]; then
    echo "AMBASSADOR: running with dev magic"
    diagd --dev-magic
    exit $?
fi

config_dir="${AMBASSADOR_CONFIG_BASE_DIR}/ambassador-config"
snapshot_dir="${AMBASSADOR_CONFIG_BASE_DIR}/snapshots"
diagd_flags=('--notices' "${AMBASSADOR_CONFIG_BASE_DIR}/notices.json")

# Make sure that base dir exists.
if [[ ! -d "$AMBASSADOR_CONFIG_BASE_DIR" ]]; then
    if ! mkdir -p "$AMBASSADOR_CONFIG_BASE_DIR"; then
        echo "Could not create $AMBASSADOR_CONFIG_BASE_DIR" >&2
        exit 1
    fi
fi

# Note that the envoy_config_file really is in ENVOY_DIR, rather than
# being in AMBASSADOR_CONFIG_BASE_DIR.
envoy_config_file="${ENVOY_DIR}/envoy.json"         # not a typo, see above
envoy_flags=('-c' "${ENVOY_BOOTSTRAP_FILE}")

# AMBASSADOR_DEBUG is a list of things to enable debugging for,
# separated by spaces; parse that in to an array.
read -r -d '' -a ambassador_debug <<<"$AMBASSADOR_DEBUG"
if in_array 'diagd' "${ambassador_debug[@]}"; then diagd_flags+=('--debug'); fi
if in_array 'envoy' "${ambassador_debug[@]}"; then envoy_flags+=('-l' 'debug'); fi

if in_array 'entrypoint'; then
    ENTRYPOINT_DEBUG=true

    echo "ENTRYPOINT_DEBUG enabled"
fi

if in_array 'entrypoint_trace'; then
    echo "ENTRYPOINT_TRACE enabled"

    echo 2>&1
    set -x
fi

if [[ "$1" == "--demo" ]]; then
    # This is _not_ meant to be overridden by AMBASSADOR_CONFIG_BASE_DIR.
    # It's baked into a specific location during the build process.
    config_dir="$ambassador_root/ambassador-demo-config"

    # Remember that we're running the demo in a way that we can later log
    # about...
    AMBASSADOR_DEMO_MODE=true

    # ...and remember that we mustn't try to start Kubewatch at all.
    AMBASSADOR_NO_KUBEWATCH=demo
fi

# Do we have config on the filesystem?
if [[ $(find "${config_dir}" -type f 2>/dev/null | wc -l) -gt 0 ]]; then
    echo "AMBASSADOR: using ${config_dir@Q} for configuration"
    diagd_flags+=('--config-path' "${config_dir}")

    # Don't watch for Kubernetes changes.
    if [[ -z "${AMBASSADOR_FORCE_KUBEWATCH}" ]]; then
        echo "AMBASSADOR: not watching for Kubernetes config"
        export AMBASSADOR_NO_KUBEWATCH=no_kubewatch
    fi
fi

# Start using ancient kubewatch to get our cluster ID, if we're allowed to.
# XXX Ditch this, really.
#
# We can do this unconditionally because if AMBASSADOR_CLUSTER_ID was
# set before, kubewatch sync will use it, and also because kubewatch.py
# will DTRT if Kubernetes is not available.

if ! AMBASSADOR_CLUSTER_ID=$(/usr/bin/python3 "$APPDIR/kubewatch.py" --debug); then
    echo "AMBASSADOR: could not determine cluster-id; exiting"
    exit 1
fi

export AMBASSADOR_CLUSTER_ID

echo "AMBASSADOR: starting with environment:"
echo "===="
env | grep AMBASSADOR | sort
echo "===="

mkdir -p "${snapshot_dir}"
mkdir -p "${ENVOY_DIR}"

################################################################################
# Set up job management                                                        #
################################################################################

pids=()

launch() {
    cmd="$1"
    shift

    echo "AMBASSADOR: launching worker process '${cmd}': ${*@Q}"

    # We do this 'eval' instead of just
    #     "$@" &
    # so that the pretty name for the job is the actual command line,
    # instead of the literal 4 characters "$@".
    eval "${@@Q} &"

    pid=$!

    pids+=("${pid}:${cmd}")

    return $pid
}

handle_chld () {
    trap - CHLD
    local tmp=()

    for entry in "${pids[@]}"; do
        local pid="${entry%:*}"
        local name="${entry#*:}"

        if [ ! -d "/proc/${pid}" ]; then
            wait "${pid}"
            STATUS=$?

            echo "AMBASSADOR: $name exited: $STATUS"
            echo "AMBASSADOR: shutting down"

#            diediedie "${name}" "$STATUS"
            exit "$STATUS"
        else
            echo "AMBASSADOR: $name still running"
            tmp+=("${entry}")
        fi
    done

    # Reset $pids...
    pids=(${tmp[@]})

    trap "handle_chld" CHLD
}

set -m # We need this in order to trap on SIGCHLD

trap 'handle_chld' CHLD # Notify when a job status changes

trap 'echo "Received SIGINT (Control-C?); shutting down"; jobs -p | xargs -r kill --' INT

################################################################################
# WORKER: DEMO                                                                 #
################################################################################
if [[ -n "$AMBASSADOR_DEMO_MODE" ]]; then
    launch "demo-auth" env PORT=5050 python3 demo-services/auth.py
    launch "demo-qotm" python3 demo-services/qotm.py
fi

################################################################################
# WORKER: AMBEX                                                                #
################################################################################
if [[ -z "${DIAGD_ONLY}" ]]; then
    launch "ambex" ambex -ads 8003 "${ENVOY_DIR}"
    ambex_pid=$?

    diagd_flags+=('--kick' "kill -HUP $$")
else
    diagd_flags+=('--no-checks' '--no-envoy')
fi

# Once Ambex is running, we can set up ADS management

envoy_pid=
demo_chimed=

kick_ads() {
    if [ -n "$DIAGD_ONLY" ]; then
        echo "kick_ads: ignoring kick since in diagd-only mode."
    else
        if [ -n "${envoy_pid}" ]; then
            if ! kill -0 "${envoy_pid}"; then
                envoy_pid=
            fi
        fi

        if [ -z "${envoy_pid}" ]; then
            # Envoy isn't running. Start it.
            launch "envoy" envoy "${envoy_flags[@]}"

            envoy_pid=$?

            echo "KICK: started Envoy as PID $envoy_pid"
        fi

        # Once envoy is running, poke Ambex.

        if [ -n "$ENTRYPOINT_DEBUG" ]; then
            echo "KICK: kicking ambex"
        fi

        kill -HUP "$ambex_pid"

        if [ -n "$AMBASSADOR_DEMO_MODE" -a -z "$demo_chimed" ]; then
            # Wait for Envoy...
            tries_left=10
            delay=1

            while (( tries_left > 0 )); do
                echo "AMBASSADOR: pinging envoy ($tries_left)..."

                status=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8001/ready)

                if [ "$status" = "200" ]; then
                    break
                fi

                tries_left=$(( tries_left - 1 ))
                sleep $delay
                delay=$(( delay * 2 ))
                if (( delay > 10 )); then delay=5; fi
            done
            if (( tries_left <= 0 )); then
                echo "AMBASSADOR: giving up on envoy and hoping for the best..."
            else
                echo "AMBASSADOR: envoy running"
            fi

            echo "AMBASSADOR DEMO RUNNING"
            demo_chimed=yes
        fi
    fi
}

# On SIGHUP, kick ADS
trap 'kick_ads' HUP

################################################################################
# WORKER: DIAGD                                                                #
################################################################################
# We can't start Envoy until the initial config happens, which means that diagd has to start it.

launch "diagd" diagd \
       "${snapshot_dir}" \
       "${ENVOY_BOOTSTRAP_FILE}" \
       "${envoy_config_file}" \
       "${diagd_flags[@]}"

# Wait for diagd to start
tries_left=10
delay=1
while (( tries_left > 0 )); do
    echo "AMBASSADOR: pinging diagd ($tries_left)..."

    status=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8877/_internal/v0/ping)

    if [ "$status" = "200" ]; then
        break
    fi

    tries_left=$(( tries_left - 1 ))
    sleep $delay
    delay=$(( delay * 2 ))
    if (( delay > 10 )); then delay=5; fi
done
if (( tries_left <= 0 )); then
    echo "AMBASSADOR: giving up on diagd and hoping for the best..."
else
    echo "AMBASSADOR: diagd running"
fi


################################################################################
# WORKER: KUBEWATCH                                                            #
################################################################################
if [[ -z "${AMBASSADOR_NO_KUBEWATCH}" ]]; then
    KUBEWATCH_SYNC_KINDS="-s service"

    if [ ! -f "${AMBASSADOR_CONFIG_BASE_DIR}/.ambassador_ignore_crds" ]; then
        KUBEWATCH_SYNC_KINDS="$KUBEWATCH_SYNC_KINDS -s AuthService -s Mapping -s Module -s RateLimitService -s TCPMapping -s TLSContext -s TracingService"
    fi

    if [ ! -f "${AMBASSADOR_CONFIG_BASE_DIR}/.ambassador_ignore_crds_2" ]; then
        KUBEWATCH_SYNC_KINDS="$KUBEWATCH_SYNC_KINDS -s ConsulResolver -s KubernetesEndpointResolver -s KubernetesServiceResolver"
    fi

    AMBASSADOR_FIELD_SELECTOR_ARG=""
    if [ -n "$AMBASSADOR_FIELD_SELECTOR" ] ; then
	    AMBASSADOR_FIELD_SELECTOR_ARG="--fields $AMBASSADOR_FIELD_SELECTOR"
    fi

    AMBASSADOR_LABEL_SELECTOR_ARG=""
    if [ -n "$AMBASSADOR_LABEL_SELECTOR" ] ; then
	    AMBASSADOR_LABEL_SELECTOR_ARG="--labels $AMBASSADOR_LABEL_SELECTOR"
    fi

    if [ "${AMBASSADOR_KNATIVE_SUPPORT}" = true ]; then
        KUBEWATCH_SYNC_KINDS="$KUBEWATCH_SYNC_KINDS -s ClusterIngress"
    fi

    launch "watt" /ambassador/watt \
           --port 8002 \
           ${AMBASSADOR_SINGLE_NAMESPACE:+ --namespace "${AMBASSADOR_NAMESPACE}" } \
           --notify 'sh /ambassador/post_watt.sh' \
           ${KUBEWATCH_SYNC_KINDS} \
           ${AMBASSADOR_FIELD_SELECTOR_ARG} \
           ${AMBASSADOR_LABEL_SELECTOR_ARG} \
           --watch /ambassador/watch_hook.py
fi

################################################################################
# Wait for one worker to quit, then kill the others                            #
################################################################################

echo "AMBASSADOR: waiting"
echo "PIDS: $pids"

while true; do
    wait
    echo "-ping-"
done

#ambassador_exit 0
exit 0
