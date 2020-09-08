#!/bin/bash

# Choose colors carefully. If they don't work on both a black
# background and a white background, pick other colors (so white,
# yellow, and black are poor choices).
export RED='\033[1;31m'
export GRN='\033[1;32m'
export BLU='\033[1;34m'
export CYN='\033[1;36m'
export END='\033[0m'

# Check to see that an environment variable has been provided exit with an error message if it is
# not available.
#
# Usage: require <varname>
require() {
    if [ -z "${!1}" ]; then
        echo "please set the $1 environment variable" 2>&1
        exit 1
    fi
}

# Wait for a kubernetes service to have an external IP address, and echo that address.
#
# Usage: wait_for_ip <namespace> <service_name>
#
# This can be used as follows in scripts:
#
#   # Grab the external ip address of the ambassador service in the ambassador namespace:
#   AMBASSADOR_IP="$(wait_for_ip ambassador ambassador)"
#
wait_for_ip() ( # use a subshell so the set +x is local to the function
    { set +x; } 2>/dev/null # make the set +x be quiet
    local external_ip=""
    while true; do
        external_ip=$(kubectl get svc -n "$1" "$2" --template="{{range .status.loadBalancer.ingress}}{{.ip}}{{end}}")
        if [ -z "$external_ip" ]; then
            echo "Waiting for external IP..." 1>&2
            sleep 10
        else
            break
        fi
    done
    echo "$external_ip"
)

# Wait for a URL to return a 200 instead of an error.
#
# Usage: wait_for_url <URL>
wait_for_url() ( # use a subshell so the set +x is local to the function
    { set +x; } 2>/dev/null # make the set +x be quiet
    local status=""
    while true; do
        status=$(curl -k -sL -w "%{http_code}" -o /dev/null "$@")
        if [ "$status" != "200" ]; then
            echo "Got $status, waiting for 200..." 1>&2
            sleep 10
        else
            break
        fi
    done
)

# Robustly wait for a deployment to be ready. This includes checking for crashloop backoffs and
# handling the case where a deployment was already in the ready state, but an updated image has been
# applied.
#
# Usage: wait_for_deployment <namespace> <deployment_name>
wait_for_deployment() ( # use a subshell so the set +x is local to the function
    { set +x; } 2>/dev/null # make the set +x be quiet
    # Check deployment rollout status every 10 seconds (max 10 minutes) until complete.
    local attempts=0
    while true; do
        if kubectl rollout status "deployment/${2}" -n "$1" 1>&2; then
            break
        else
            CRASHING="$(crashLoops ambassador)"
            if [ -n "${CRASHING}" ]; then
                echo "${CRASHING}" 1>&2
                return 1
            fi
        fi

        if [ $attempts -eq 60 ]; then
            echo "deploy timed out" 1>&2
            return 1
        fi

        attempts=$((attempts + 1))
        sleep 10
    done
)

# helper used by wait_for_deployment
crashLoops() ( # use a subshell so the set +x is local to the function
    { set +x; } 2>/dev/null # make the set +x be quiet
    # shellcheck disable=SC2016
    kubectl get pods -n "$1" -o 'go-template={{range $pod := .items}}{{range .status.containerStatuses}}{{if .state.waiting}}{{$pod.metadata.name}} {{.state.waiting.reason}}{{"\n"}}{{end}}{{end}}{{end}}' | grep CrashLoopBackOff
)

# Start acquisition of a kubeception cluster. This will start spinning up a kubeception cluster and
# return immediately. This takes exactly the same arguments that `get_cluster` does. See the
# get_cluster documentation for details. Use `await_cluster` to wait until the cluster is ready for
# use.
#
# This command requires the KUBECEPTION_TOKEN env var to be set.
start_cluster() {
    local kubeconfig timeout profile
    kubeconfig=${1}
    timeout=${2:-3600}
    profile=${3:-default}
    if [ -e "${kubeconfig}" ]; then
        echo "cannot get cluster, kubeconfig ${kubeconfig} exists" 1>&2
        return 1
    fi
    curl -s -H "Authorization: bearer ${KUBECEPTION_TOKEN}" "https://sw.bakerstreet.io/kubeception/api/klusters/ci-?generate=true&timeoutSecs=${timeout}&profile=${profile}" -X PUT > "${kubeconfig}"
    printf "${BLU}Acquiring cluster:\n==${END}\n" 1>&2
    cat "${kubeconfig}" 1>&2
    printf "${BLU}==${END}\n" 1>&2
}

# Wait for a kubeception cluster to be ready for use. This must be used in combination with a prior
# invocation of `start_cluster`. The `await_cluster` call should be passed exactly the same
# arguments that were passed to `start_cluster`. See `start_cluster` and `get_cluster` for details.
#
# This command requires the KUBECEPTION_TOKEN env var to be set.
await_cluster() {
    local kubeconfig name kconfurl
    kubeconfig=${1}
    name="$(head -1 "${kubeconfig}" | cut -c2-)"
    kconfurl="https://sw.bakerstreet.io/kubeception/api/klusters/${name}"
    wait_for_url "$kconfurl" -H "Authorization: bearer ${KUBECEPTION_TOKEN}"
    curl -s -H "Authorization: bearer ${KUBECEPTION_TOKEN}" "$kconfurl" -o "${kubeconfig}"
}

# Get a kubeception cluster.
#
# Usage: get_cluster <path_to_kubeconfig> [ <lifespan_in_seconds> [ <profile> ] ]
#
# You must provide a path to where you would like the kubeconfig file for the newly acquired cluster
# to be. The get_cluster command will refuse to overwrite an existing kubeconfig file and report an
# error in that case.
#
# The default lifespan is one hour (3600 seconds), and the default profile is "default".
#
# The get_cluster command will block until the cluster is available, however it is syntactic sugar
# for `start_cluster` followed by `await_cluster`. Using `start_cluster` and `await_cluster` might
# provide for a more optimized CI usage since you can e.g. run `start_cluster` (which will return
# immediately), then build your code, then call `await_cluster` before running your tests, just in
# case your build finished before the cluster was ready.
#
# The arguments for `start_cluster` and `await_cluster` are identical to the arguments for
# `get_cluster`.
#
# This command requires the KUBECEPTION_TOKEN env var to be set.
get_cluster() {
    start_cluster "$@"
    await_cluster "$@"
}

# Release a kubeception cluster when done.
#
# Usage: del_cluster <path_to_kubeconfig>
#
# You must provide a path to the exact same kubeconfig that was created by a call to `get_cluster`
# or `start_cluster`. This command will delete the kubeconfig file once the cluster has been
# deleted.
#
# This command requires the KUBECEPTION_TOKEN env var to be set.
del_cluster() {
    local kubeconfig name
    kubeconfig=${1}
    name="$(head -1 "${kubeconfig}" | cut -c2-)"
    curl -s -H "Authorization: bearer ${KUBECEPTION_TOKEN}" "https://sw.bakerstreet.io/kubeception/api/klusters/${name}" -X DELETE
    rm -f "${kubeconfig}"
}
