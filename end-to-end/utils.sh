if [ -z "$ROOT" ]; then
    echo "ROOT must be set to the root of the end-to-end tests" >&2
    exit 1
fi

if [ -n "$MACHINE_READABLE" ]; then
    LINE_END="\n"
else
    LINE_END="\r"
fi

step () {
    echo "==== $@"
}

initialize_cluster () {
    for namespace in $(kubectl get namespaces | egrep -v '^(NAME|kube-)' | awk ' { print $1 }'); do
        echo "Deleting everything in $namespace..."

        if [ "$namespace" = "default" ]; then
            kubectl delete pods,secrets,services,deployments,configmaps --all
        else
            kubectl delete namespace "$namespace"
        fi
    done
}

cluster_ip () {
    IP=$(kubectl get nodes -ojsonpath="{.items[0].status.addresses[?(@.type==\"ExternalIP\")].address}")

    if [ -z "$IP" ]; then
        IP=$(kubectl cluster-info | fgrep master | python -c 'import sys; print(sys.stdin.readlines()[0].split()[5].split(":")[1].lstrip("/"))')
    fi

    echo "$IP"
}

service_port() {
    instance=${2:-0}

    kubectl get services "$1" -ojsonpath="{.spec.ports[$instance].nodePort}"
}

demotest_pod() {
    kubectl get pods -l run=demotest -o 'jsonpath={.items[0].metadata.name}'
}

wait_for_pods () {
    namespace=${1:-default}
    attempts=60
    running=

    while [ $attempts -gt 0 ]; do
        # pending=$(kubectl --namespace $namespace get pod -o json | grep phase | grep -c -v Running)
        pending=$(kubectl --namespace $namespace describe pods | grep '^Status:' | grep -c -v Running)

        if [ $pending -eq 0 ]; then
            printf "Pods running.              \n"
            running=YES
            break
        fi

        printf "try %02d: %d not running${LINE_END}" $attempts $pending
        attempts=$(( $attempts - 1 ))
        sleep 2
    done

    if [ -z "$running" ]; then
        echo 'Some pods have yet to start?' >&2
        exit 1
    fi
}

wait_for_ready () {
    baseurl=${1}
    attempts=60
    ready=

    while [ $attempts -gt 0 ]; do
        OK=$(curl -k $baseurl/ambassador/v0/check_ready 2>&1 | grep -c 'readiness check OK')

        if [ $OK -gt 0 ]; then
            printf "ambassador ready           \n"
            ready=YES
            break
        fi

        printf "try %02d: not ready${LINE_END}" $attempts
        attempts=$(( $attempts - 1 ))
        sleep 2
    done

    if [ -z "$ready" ]; then
        echo 'Ambassador not yet ready?' >&2
        kubectl get pods >&2
        exit 1
    fi
}

wait_for_extauth_running () {
    baseurl=${1}
    attempts=60
    ready=

    while [ $attempts -gt 0 ]; do
        OK=$(curl -k -s $baseurl/example-auth/ready | egrep -c '^OK ')

        if [ $OK -gt 0 ]; then
            printf "extauth ready              \n"
            ready=YES
            break
        fi

        printf "try %02d: not ready${LINE_END}" $attempts
        attempts=$(( $attempts - 1 ))
        sleep 5
    done

    if [ -z "$ready" ]; then
        echo 'extauth not yet ready?' >&2
        exit 1
    fi
}

wait_for_extauth_enabled () {
    baseurl=${1}
    attempts=60
    enabled=

    while [ $attempts -gt 0 ]; do
        OK=$(curl -k -s $baseurl/ambassador/v0/diag/?json=true | jget.py /filters/0/name 2>&1 | egrep -c 'extauth')

        if [ $OK -gt 0 ]; then
            printf "extauth enabled            \n"
            enabled=YES
            break
        fi

        printf "try %02d: not enabled${LINE_END}" $attempts
        attempts=$(( $attempts - 1 ))
        sleep 5
    done

    if [ -z "$enabled" ]; then
        echo 'extauth not yet enabled?' >&2
        exit 1
    fi
}

wait_for_demo_weights () {
    attempts=60
    routed=

    while [ $attempts -gt 0 ]; do
        if checkweights.py "$@"; then
            routed=YES
            break
        fi

        printf "try %02d: misweighted${LINE_END}" $attempts
        attempts=$(( $attempts - 1 ))
        sleep 5
    done

    if [ -z "$routed" ]; then
        echo 'weights still not correct?' >&2
        exit 1
    fi
}

check_diag () {
    baseurl=$1
    index=$2
    desc=$3

    sleep 5

    rc=1

    curl -k -s ${baseurl}/ambassador/v0/diag/?json=true | jget.py /routes > check-$index.json

    if ! cmp -s check-$index.json diag-$index.json; then
        echo "check_diag $index: mismatch for $desc"

        if diag-diff.sh $index; then
            diag-fix.sh $index
            rc=0
        fi
    else
        echo "check_diag $index: OK"
        rc=0
    fi

    return $rc
}

istio_running () {
    kubectl get service istio-mixer >/dev/null 2>&1
}

ambassador_pod () {
    kubectl get pod -l app=ambassador -o jsonpath='{.items[0].metadata.name}'
}

# ISTIOHOME=${ISTIOHOME:-${HERE}/istio-0.1.6}

# source ${ISTIOHOME}/istio.VERSION

# if [ \( "$1" = "--delete" \) -o \( "$1" = "-d" \) ]; then
#     ACTION="delete"
#     HRACTION="Tearing down"
#     shift
# else
#     ACTION="apply"
#     HRACTION="Setting up"
# fi

# KUBEDIR=${HERE}/kube
