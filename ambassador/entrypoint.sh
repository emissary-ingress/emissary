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
export PYTHON_EGG_CACHE="${PYTHON_EGG_CACHE:-$APPDIR}/.cache"
export PYTHONUNBUFFERED=true

config_dir="${AMBASSADOR_CONFIG_BASE_DIR}/ambassador-config"
snapshot_dir="${AMBASSADOR_CONFIG_BASE_DIR}/snapshots"
diagd_flags=('--notices' "${AMBASSADOR_CONFIG_BASE_DIR}/notices.json")

# Note that the envoy_config_file really is in ENVOY_DIR, rather than
# being in AMBASSADOR_CONFIG_BASE_DIR.
envoy_config_file="${ENVOY_DIR}/envoy.json"         # not a typo, see above
envoy_flags=('-c' "${ENVOY_BOOTSTRAP_FILE}")

# AMBASSADOR_DEBUG is a list of things to enable debugging for,
# separated by spaces; parse that in to an array.
read -r -d '' -a ambassador_debug <<<"$AMBASSADOR_DEBUG"
if in_array 'diagd' "${ambassador_debug[@]}"; then diagd_flags+=('--debug'); fi
if in_array 'envoy' "${ambassador_debug[@]}"; then envoy_flags+=('-l' 'debug'); fi

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
launch() {
    echo "AMBASSADOR: launching worker process: ${*@Q}"
    # We do this 'eval' instead of just
    #     "$@" &
    # so that the pretty name for the job is the actual command line,
    # instead of the literal 4 characters "$@".
    eval "${@@Q} &"
}
set -m # We need this in order to trap on SIGCHLD
trap 'jobs -n' CHLD # Notify when a job status changes

trap 'echo "Received SIGINT (Control-C?); shutting down"; jobs -p | xargs -r kill --' INT

################################################################################
# WORKER: DEMO                                                                 #
################################################################################
if [[ -n "$AMBASSADOR_DEMO_MODE" ]]; then
    launch env PORT=5050 python3 demo-services/auth.py
    launch python3 demo-services/qotm.py
fi

################################################################################
# WORKER: AMBEX                                                                #
################################################################################
if [[ -z "${DIAGD_ONLY}" ]]; then
    echo "AMBASSADOR: starting ads"
    launch ambex -ads 8003 "${ENVOY_DIR}"
    ambex_pid="$!"
    diagd_flags+=('--kick' "/ambassador/kick_ads.sh ${ambex_pid@Q} ${envoy_flags[*]@Q}")
else
    diagd_flags+=('--no-checks' '--no-envoy')
fi

################################################################################
# WORKER: DIAGD                                                                #
################################################################################
# We can't start Envoy until the initial config happens, which means that diagd has to start it.
echo "AMBASSADOR: starting diagd"
launch diagd \
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
    if [ ! -f .ambassador_ignore_crds ]; then
        KUBEWATCH_SYNC_KINDS="$KUBEWATCH_SYNC_KINDS -s AuthService -s ConsulResolver -s KubernetesEndpointResolver -s KubernetesServiceResolver -s Mapping -s Module -s RateLimitService -s TCPMapping -s TLSContext -s TracingService"
    fi

    launch /ambassador/watt \
           --port 8002 \
           ${AMBASSADOR_SINGLE_NAMESPACE:+ --namespace "${AMBASSADOR_NAMESPACE}" } \
           --notify 'sh /ambassador/post_watt.sh' \
           ${KUBEWATCH_SYNC_KINDS} \
           --watch /ambassador/watch_hook.py
fi

################################################################################
# Wait for one worker to quit, then kill the others                            #
################################################################################
echo "AMBASSADOR: waiting"
echo "AMBASSADOR: worker PIDs:" $(jobs -p)

if [[ -n "$AMBASSADOR_DEMO_MODE" ]]; then
    echo "AMBASSADOR DEMO RUNNING"
fi

wait -n
r=$?
echo 'AMBASSADOR: one of the worker processes has exited; shutting down the others'
while test -n "$(jobs -p)"; do
    jobs -p | xargs -r kill --
    wait -n
done
echo 'AMBASSADOR: all worker processes have exited'
if [[ -n "$AMBASSADOR_EXIT_DELAY" ]]; then
    echo "AMBASSADOR: sleeping for debug"
    sleep $AMBASSADOR_EXIT_DELAY
fi
exit $r
