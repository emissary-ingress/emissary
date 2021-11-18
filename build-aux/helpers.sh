#!/bin/bash

# Choose colors carefully. If they don't work on both a black
# background and a white background, pick other colors (so white,
# yellow, and black are poor choices).
export RED='\033[1;31m'
export GRN='\033[1;32m'
export BLU='\033[1;34m'
export CYN='\033[1;36m'
export END='\033[0m'

require() {
    if [ -z "${!1}" ]; then
        echo "please set the $1 environment variable" 2>&1
        exit 1
    fi
}

kubectl() {
    if ! test -f tools/bin/kubectl; then
        make tools/bin/kubectl >&2
    fi
    tools/bin/kubectl "$@"
}

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

curl_retry() (
    { set +x; } 2>/dev/null # make the set +x be quiet
    local tries=0
    local retry_count=$1
    echo "retry_count ${retry_count}"
    shift 1
    local ok_status_codes=$1
    echo "ok_status_codes ${ok_status_codes}"
    shift 1
    local abort_status_codes=$1
    echo "abort status codes ${abort_status_codes}"
    shift 1
    while [ "${retry_count}" -lt 0 ] || [ "$tries" -lt "${retry_count}" ] ; do
        local code=$(curl -v --retry 100 --max-time 120 --retry-connrefused -skL -w "%{http_code}" "$@")
        local curlStatus="$?"
        echo $curlStatus
        if [ "$curlStatus" -ne "56" ] && [ "$curlStatus" -ne "18" ] ; then
            if [ "${ok_status_codes}" == "" ] ; then
                return 0
            fi
            for ok_code in ${ok_status_codes//,/ } ; do
                if [ "${ok_code}" -eq "${code}" ] ; then
                    echo "Got $code, ready!" 1>&2
                    return 0
                fi
            done
            for bad_code in ${abort_status_codes//,/ } ; do
                if [ "${bad_code}" -eq "${code}" ] ; then
                    echo "Got $code, aborting" 1>&2
                    return 1
                fi
            done
        fi
        echo "Curl exited with code $curlStatus and status code ${code}, sleeping for 10 seconds and then retrying... (tries=$tries)" 1>&2
        sleep 10
        tries=$((tries+1))
    done
    return 1
)

wait_for_url_output() ( # use a subshell so the set +x is local to the function
    { set +x; } 2>/dev/null # make the set +x be quiet
    local status=""
    local output="${1}"
    shift 1
    while true; do
        status=$(curl --retry 100 --retry-connrefused -k -sL -w "%{http_code}" -o "${output}" "$@")
        if [ "$status" == "400" ]; then
            echo "Got $status, aborting" 1>&2
            exit 1
        elif [ "$status" != "200" ]; then
            echo "Got $status, waiting for 200..." 1>&2
            sleep 10
        else
            echo "Ready!" 1>&2
            break
        fi
    done
)

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

wait_for_kubeconfig() ( # use a subshell so the set +x is local to the function
    { set +x; } 2>/dev/null # make the set +x be quiet
    local attempts=0
    local kubeconfig="${1}"
    while true; do
        if kubectl --kubeconfig ${kubeconfig} -n default get service kubernetes; then
            break
        fi

        if [ $attempts -eq 60 ]; then
            echo "kubeconfig ${kubeconfig} timed out" 1>&2
            return 1
        fi
        attempts=$((attempts + 1))
        sleep 10
    done
)

crashLoops() ( # use a subshell so the set +x is local to the function
    { set +x; } 2>/dev/null # make the set +x be quiet
    # shellcheck disable=SC2016
    kubectl get pods -n "$1" -o 'go-template={{range $pod := .items}}{{range .status.containerStatuses}}{{if .state.waiting}}{{$pod.metadata.name}} {{.state.waiting.reason}}{{"\n"}}{{end}}{{end}}{{end}}' | grep CrashLoopBackOff
)

start_cluster() {
    local kubeconfig timeout profile retries
    kubeconfig=${1}
    timeout=${2:-3600}
    profile=${3:-default}
    version=${4:-1.19}
    retries=2
    if [ -e "${kubeconfig}" ]; then
        echo "cannot get cluster, kubeconfig ${kubeconfig} exists" 1>&2
        return 1
    fi
    klusterurl="https://sw.bakerstreet.io/kubeception/api/klusters/ci-?generate=true&timeoutSecs=${timeout}&profile=${profile}&version=${version}"
    printf "${BLU}Acquiring cluster with K8s version ${version}:\n==${END}\n" 1>&2
    curl_retry $retries "200,425" "" -o "${kubeconfig}" -H "Authorization: bearer ${KUBECEPTION_TOKEN}" "${klusterurl}" -X PUT
    ret="$?"
    if [ "${ret}" -ne "0" ] ; then
        echo "Unable to aquire cluster, exiting" 1>&2
        return ${ret}
    fi
    cat "${kubeconfig}" 1>&2
    printf "${BLU}==${END}\n" 1>&2
}

await_cluster() {
    local kubeconfig name kconfurl
    kubeconfig=${1}
    name="$(head -1 "${kubeconfig}" | cut -c2-)"
    kconfurl="https://sw.bakerstreet.io/kubeception/api/klusters/${name}"
	# 100*10s == a little over 15 min. if the kluster isn't ready at that point,
	# its probably never going ready
	curl_retry 100 "200" "400" -o "${kubeconfig}" "$kconfurl" -H "Authorization: bearer ${KUBECEPTION_TOKEN}"
	ret=$?
	if [ "${ret}" -ne "0" ] ; then
		echo "Failed waiting for cluster to come up" 1>&2
		return $ret
	fi
    printf "${BLU}Cluster ${name} acquired:\n==${END}\n" 1>&2
    cat "${kubeconfig}" 1>&2
    printf "${BLU}==${END}\n" 1>&2
}

get_cluster() {
    start_cluster "$@"
    if [ "$?" != "0" ] ; then
        return 1
    fi
    await_cluster "$@"
}

del_cluster() {
    local kubeconfig name
    kubeconfig=${1}
    name="$(head -1 "${kubeconfig}" | cut -c2-)"
    curl -s -H "Authorization: bearer ${KUBECEPTION_TOKEN}" "https://sw.bakerstreet.io/kubeception/api/klusters/${name}" -X DELETE
    rm -f "${kubeconfig}"
}
