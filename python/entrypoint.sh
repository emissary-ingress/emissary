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

# THE DEFAULT BOOT SEQUENCE IS NOW entrypoint.go. HOWEVER, we'll stick
# with entrypoint.sh when the --dev-magic parameter is present. This
# is currently used only for test_scout.py.  This is a BRUTAL HACK.

if [ "$1" != "--dev-magic" ]; then
  exec busyambassador entrypoint "$@"   # See comment above.
fi

DEVMAGIC=yes

# If we are here, define AMBASSADOR_FAST_RECONFIGURE, to make
# _absolutely certain_ that diagd's localhost checks are in sync with
# what's actually running...
export AMBASSADOR_FAST_RECONFIGURE=false

ENTRYPOINT_DEBUG=

log () {
    local now

    now=$(date +"%Y-%m-%d %H:%M:%S")
    echo "${now} AMBASSADOR INFO ${@}" >&2
}

debug () {
    local now

    if [ -n "$ENTRYPOINT_DEBUG" ]; then
        now=$(date +"%Y-%m-%d %H:%M:%S")
        echo "${now} AMBASSADOR DEBUG ${@}" >&2
    fi
}

wait_for_url () {
    local name url tries_left delay status

    name="$1"
    url="$2"

    tries_left=10
    delay=1

    while (( tries_left > 0 )); do
        debug "pinging $name ($tries_left)..."

        status=$(curl -s -o /dev/null -w "%{http_code}" $url)

        if [ "$status" = "200" ]; then
            break
        fi

        tries_left=$(( tries_left - 1 ))
        sleep $delay
        delay=$(( delay * 2 ))
        if (( delay > 10 )); then delay=5; fi
    done

    if (( tries_left <= 0 )); then
        log "giving up on $name and hoping for the best..."
    else
        log "$name running"
    fi
}

################################################################################
# CONFIG PARSING                                                               #
################################################################################

ambassador_root="/ambassador"

export LC_ALL=C.UTF-8
export LANG=C.UTF-8

# If we have an AGENT_SERVICE, but no AMBASSADOR_ID, force AMBASSADOR_ID
# from the AGENT_SERVICE.

if [ -z "$AMBASSADOR_ID" -a -n "$AGENT_SERVICE" ]; then
    export AMBASSADOR_ID="intercept-${AGENT_SERVICE}"
    log "Intercept: set AMBASSADOR_ID to $AMBASSADOR_ID"
fi

export AMBASSADOR_NAMESPACE="${AMBASSADOR_NAMESPACE:-default}"
export AMBASSADOR_CONFIG_BASE_DIR="${AMBASSADOR_CONFIG_BASE_DIR:-$ambassador_root}"
export ENVOY_DIR="${AMBASSADOR_CONFIG_BASE_DIR}/envoy"
export ENVOY_BOOTSTRAP_FILE="${AMBASSADOR_CONFIG_BASE_DIR}/bootstrap-ads.json"
export ENVOY_BASE_ID="${AMBASSADOR_ENVOY_BASE_ID:-0}"

export APPDIR="${APPDIR:-$ambassador_root}"

# If we don't set PYTHON_EGG_CACHE explicitly, /.cache is set by
# default, which fails when running as a non-privileged user
export PYTHON_EGG_CACHE="${PYTHON_EGG_CACHE:-$AMBASSADOR_CONFIG_BASE_DIR}/.cache"
export PYTHONUNBUFFERED=true

if [ -n "$DEVMAGIC" ]; then
    log "running with dev magic"
    diagd --dev-magic
    exit $?
fi

config_dir="${AMBASSADOR_CONFIG_BASE_DIR}/ambassador-config"
snapshot_dir="${AMBASSADOR_CONFIG_BASE_DIR}/snapshots"
diagd_flags=('--notices' "${AMBASSADOR_CONFIG_BASE_DIR}/notices.json")

# Make sure that $HOME exists.
#
# Note that this might end up with different permissions than the Golang
# side of the world. Ambassador doesn't really care; 0700 is all we need.
if [[ ! -d "$HOME" ]]; then
    if ! mkdir -p "$HOME"; then
        log "Could not create $HOME"
        exit 1
    fi
fi

# Make sure that base dir exists.
if [[ ! -d "$AMBASSADOR_CONFIG_BASE_DIR" ]]; then
    if ! mkdir -p "$AMBASSADOR_CONFIG_BASE_DIR"; then
        log "Could not create $AMBASSADOR_CONFIG_BASE_DIR" >&2
        exit 1
    fi
fi

# Note that the envoy_config_file really is in ENVOY_DIR, rather than
# being in AMBASSADOR_CONFIG_BASE_DIR.
envoy_config_file="${ENVOY_DIR}/envoy.json"         # not a typo, see above
envoy_flags=('-c' "${ENVOY_BOOTSTRAP_FILE}" "--base-id" "${ENVOY_BASE_ID}")
envoy_logging=('-l' 'error')

if [ -n "$AGENT_SERVICE" ]; then
    # We are the intercept agent. Force Envoy's drain time very low.
    envoy_flags+=( "--drain-time-s" "1" )
else
    # We are not the intercept agent, so leave Envoy's drain time much higher.
    drain_time="${AMBASSADOR_DRAIN_TIME:-600}"
    debug "Using drain_time $drain_time"
    envoy_flags+=( "--drain-time-s" "$drain_time" )
fi

# AMBASSADOR_DEBUG is a list of things to enable debugging for,
# separated by spaces; parse that in to an array.
read -r -d '' -a ambassador_debug <<<"$AMBASSADOR_DEBUG"
for item in "${ambassador_debug[@]}"; do
    case "$item" in
        diagd)
            debug 'AMBASSADOR_DEBUG: `diagd --debug` enabled'
            diagd_flags+=('--debug')
            ;;
        envoy)
            debug 'AMBASSADOR_DEBUG: `envoy -l debug` enabled'
            envoy_logging=('-l' 'debug')
            ;;
        entrypoint)
            debug "AMBASSADOR_DEBUG: ENTRYPOINT_DEBUG enabled"
            ENTRYPOINT_DEBUG=true
            ;;
        entrypoint_trace)
            debug "AMBASSADOR_DEBUG: ENTRYPOINT_TRACE enabled"
            echo 2>&1
            set -x
            ;;
        *)
            debug "AMBASSADOR_DEBUG: unknown item: ${item@Q}"
            ;;
    esac
done

envoy_flags+=( "${envoy_logging[@]}" )

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
    log "using ${config_dir@Q} for configuration"
    diagd_flags+=('--config-path' "${config_dir}")

    # Don't watch for Kubernetes changes.
    if [[ -z "${AMBASSADOR_FORCE_KUBEWATCH}" ]]; then
        log "not watching for Kubernetes config"
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
    log "could not determine cluster-id; exiting"
    exit 1
fi

export AMBASSADOR_CLUSTER_ID

log "starting with environment:"
log "===="
env | grep AMBASSADOR | sort
log "===="

mkdir -p "${snapshot_dir}"
mkdir -p "${ENVOY_DIR}"

################################################################################
# Termination funcions                                                         #
################################################################################

ambassador_exit() {
    RC=${1:-1}

    if [ -n "$AMBASSADOR_EXIT_DELAY" ]; then
        log "sleeping before shutdown ($RC)"
        sleep $AMBASSADOR_EXIT_DELAY
    fi

    log "killing extant processes"
    jobs -p | xargs -r kill --

    log "shutting down ($RC)"
    exit $RC
}

diediedie() {
    NAME=$1
    STATUS=$2

    if [ $STATUS -eq 0 ]; then
        log "$NAME claimed success, but exited \?\?\?\?"
    else
        log "$NAME exited with status $STATUS"
    fi

    ambassador_exit 1
}

################################################################################
# Set up job management                                                        #
################################################################################
#
# We can't completely rely on Bash job control for this, because our SIGHUP
# trap will trigger job control to think that something has exited! So we need
# to explicitly trap SIGCHLD and make sure that the thing that exited isn't one
# of our _important_ processes.

declare -A pids # associative array of cmd:pid

launch() {
    local cmd args pid

    cmd="$1"    # this is a human-readable name used only for logging.
    shift
    args="${@@Q}"

    log "launching worker process '${cmd}': ${args}"

    # We do this 'eval' instead of just
    #     "$@" &
    # so that the pretty name for the job is the actual command line,
    # instead of the literal 4 characters "$@".
    eval "${args} &"

    pid=$!

    pids[$cmd]=$pid

    log "${cmd} is PID ${pid}"

    if [ -n "$ENTRYPOINT_DEBUG" ]; then
        for K in "${!pids[@]}"; do
            echo "AMBASSADOR pids $K --- ${pids[$K]}"
        done
    fi
}

handle_chld () {
    trap - CHLD

    local cmd pid status

    for cmd in "${!pids[@]}"; do
        pid=${pids[$cmd]}

        if [ ! -d "/proc/${pid}" ]; then
            wait "${pid}"
            status=$?

            pids[$cmd]=
            diediedie "${cmd}" "$status"
        else
            if [ -n "$ENTRYPOINT_DEBUG" ]; then
                debug "$cmd still running"
            fi
        fi
    done

    trap "handle_chld" CHLD
}

set -m # We need this in order to trap on SIGCHLD

trap 'handle_chld' CHLD # Notify when a job status changes

trap 'log "Received SIGINT (Control-C?); shutting down"; ambassador_exit 1' INT

# Check if AMBASSADOR_DIAGD_BIND_ADDRESS is set, and if so, bind diagd server to that address.
if [[ -n "${AMBASSADOR_DIAGD_BIND_ADDRESS}" ]]; then
    diagd_flags+=('--host' "${AMBASSADOR_DIAGD_BIND_ADDRESS}")
fi

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
    # Keep the listen address in-sync with v2bootstrap.py
    launch "ambex" ambex --ads-listen-address=127.0.0.1:8003 "${ENVOY_DIR}"

    diagd_flags+=('--kick' "kill -HUP $$")
else
    diagd_flags+=('--no-checks' '--no-envoy')
fi

# Once Ambex is running, we can set up ADS management

demo_chimed=

kick_ads() {
    if [ -n "$DIAGD_ONLY" ]; then
        debug "kick_ads: ignoring kick since in diagd-only mode."
    else
        if [ -n "${pids[envoy]}" ]; then
            if ! kill -0 "${pids[envoy]}"; then
                pids[envoy]=
            fi
        fi

        if [ -z "${pids[envoy]}" ]; then
            # Envoy isn't running. Start it.
            launch "envoy" envoy "${envoy_flags[@]}"

            log "KICK: started Envoy as PID ${pids[envoy]}"
        fi

        # Once envoy is running, poke Ambex.

        if [ -n "$ENTRYPOINT_DEBUG" ]; then
            log "KICK: kicking ambex"
        fi

        kill -HUP "${pids[ambex]}"

        if [ -n "$AMBASSADOR_DEMO_MODE" -a -z "$demo_chimed" ]; then
            # Wait for Envoy...
            wait_for_url "envoy" "http://localhost:8877/ambassador/v0/check_ready"

            log "AMBASSADOR DEMO RUNNING"
            demo_chimed=yes
        fi
    fi
}

# On SIGHUP, kick ADS
trap 'kick_ads' HUP

################################################################################
# WORKER: TRAFFIC AGENT (app-sidecar)                                          #
################################################################################
# If AGENT_SERVICE is set, we start the app-sidecar. Note that we also
# disable init-config in the ResourceFetcher when this is set!

if [[ -n "${AGENT_SERVICE}" ]]; then
    launch "app-sidecar" app-sidecar
fi

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
wait_for_url "diagd" "http://localhost:8877/_internal/v0/ping"

################################################################################
# WORKER: KUBEWATCH                                                            #
################################################################################
if [[ -z "${AMBASSADOR_NO_KUBEWATCH}" ]]; then
    watt_query_flags=(-s service)

    if [ ! -f "${AMBASSADOR_CONFIG_BASE_DIR}/.ambassador_ignore_ingress_class" ]; then
        watt_query_flags+=(-s ingressclasses)
    fi

    if [ ! -f "${AMBASSADOR_CONFIG_BASE_DIR}/.ambassador_ignore_ingress" ]; then
        watt_query_flags+=(-s ingresses)
    fi

    if [ ! -f "${AMBASSADOR_CONFIG_BASE_DIR}/.ambassador_ignore_crds" ]; then
        watt_query_flags+=(-s AuthService -s Mapping -s Module -s RateLimitService -s TCPMapping -s TLSContext -s TracingService)
    fi

    if [ ! -f "${AMBASSADOR_CONFIG_BASE_DIR}/.ambassador_ignore_crds_2" ]; then
        watt_query_flags+=(-s ConsulResolver -s KubernetesEndpointResolver -s KubernetesServiceResolver)
    fi

    if [ ! -f "${AMBASSADOR_CONFIG_BASE_DIR}/.ambassador_ignore_crds_3" ]; then
        watt_query_flags+=(-s Host -s Listener)
    fi

    if [ ! -f "${AMBASSADOR_CONFIG_BASE_DIR}/.ambassador_ignore_crds_4" ]; then
        watt_query_flags+=(-s LogService)
    fi

    if [ ! -f "${AMBASSADOR_CONFIG_BASE_DIR}/.ambassador_ignore_crds_5" ]; then
        watt_query_flags+=(-s DevPortal)
    fi

    if [ -n "$AMBASSADOR_FIELD_SELECTOR" ] ; then
	    watt_query_flags+=(--fields $AMBASSADOR_FIELD_SELECTOR)
    fi

    if [ -n "$AMBASSADOR_LABEL_SELECTOR" ] ; then
	    watt_query_flags+=(--labels $AMBASSADOR_LABEL_SELECTOR)
    fi

    if [ -n "$AMBASSADOR_SINGLE_NAMESPACE" ]; then
        watt_query_flags+=(--namespace "${AMBASSADOR_NAMESPACE}")
    fi

    launch "watt" watt \
           --listen-address="127.0.0.1:8002" \
           --notify 'python /ambassador/post_update.py --watt ' \
           --watch "python /ambassador/watch_hook.py" \
           "${watt_query_flags[@]}"
fi

################################################################################
# WORKER: extra sidecars                                                       #
################################################################################
# If AGENT_SERVICE is set, we don't do this: the intercept agent doesn't use
# any of the Edge Stack sidecars, and they just clutter up the logs doing
# nothing useful.

if [[ -z "$AGENT_SERVICE" ]]; then
    shopt -s nullglob
    for sidecar in /ambassador/sidecars/*; do
        launch "${sidecar##*/}" "$sidecar"
    done
fi

################################################################################
# Wait for one worker to quit, then kill the others                            #
################################################################################

debug "waiting"
debug "PIDS: $pids"

while true; do
    wait
    debug "-ping-"
done

ambassador_exit 2
