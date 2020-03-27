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

# set -x

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

bootstrap () {
    CLEAN_ON_SUCCESS=

    if [ "$1" == "--cleanup" ]; then
        CLEAN_ON_SUCCESS="--cleanup"
        shift
    fi

    NAMESPACE=${1:-default}

    PATH="${2}:${PATH}"
    echo "PATH set to $PATH"

    check_rbac

    initialize_namespace ${NAMESPACE}
    echo "initialized namespace $NAMESPACE"

    # Unless expand_aliases is set, no one cares about your aliases, not even bash!
    shopt -s expand_aliases
    alias kubectl="kubectl -n $NAMESPACE"

    kubectl cluster-info

    echo "bootstrapping done for test $NAMESPACE"
}

check_skip () {
    if [ -r "$ROOT/.skip-tests" ]; then
        test_name=$(basename $(pwd))

        if grep "$test_name" "$ROOT/.skip-tests" >/dev/null 2>&1; then
            echo "$test_name: skipping!"
            exit 0
        fi
    fi
}

check_rbac () {
    count=$(kubectl get clusterrole ambassador 2>/dev/null | grep -v NAME | wc -l || :)

    if [ $count -eq 0 ]; then
        kubectl apply -f $ROOT/rbac.yaml

        attempts=100
        running=

        while [ $attempts -gt 0 ]; do
            count=$(kubectl get clusterrole ambassador 2>/dev/null | grep -v NAME | wc -l || :)

            if [ $count -gt 0 ]; then
                printf "Ambassador main RBAC OK             \n"
                running=yes
                break
            fi

            printf "try %02d: waiting for RBAC\r"
            attempts=$(( $attempts - 1 ))
            sleep 2
        done

        if [ -z "$running" ]; then
            echo "could not initialize Ambassador main RBAC" >&2
            exit 1
        fi
    fi
}

initialize_cluster () {
    check_skip
    
    if [ -z "$SKIP_CHECK_CONTEXT" ]; then
        interactive_check_context
    fi

    for namespace in $(kubectl get namespaces | egrep -v '^(NAME|kube-)' | awk ' { print $1 }'); do
        echo "Deleting everything in $namespace..."

        if [ "$namespace" = "default" ]; then
            kubectl delete pods,secrets,services,deployments,configmaps --all
        else
            drop_namespace "$namespace"
        fi
    done

    wait_for_namespace_deletion
}

drop_namespace () {
    namespace="$1"

    count=$(kubectl get clusterrolebinding | grep -c "ambassador-${namespace}" || true)

    if [ $count -gt 0 ]; then
        kubectl delete clusterrolebinding ambassador-${namespace}

        attempts=100
        gone=

        while [ $attempts -gt 0 ]; do
            count=$(kubectl get clusterrolebinding | grep -c "ambassador-${namespace}" || true)

            if [ $count -eq 0 ]; then
                printf "Clusterrolebinding cleared.              \n"
                gone=yes
                break
            fi

            printf "try %02d: waiting for clusterrolebinding\r"
            attempts=$(( $attempts - 1 ))
            sleep 2
        done

        if [ -z "$gone" ]; then
            echo "could not delete clusterrolebinding ambassador-${namespace}" >&2
            exit 1
        fi
    fi

    count=$(kubectl get namespaces | grep -c "^${namespace} " || true)

    if [ $count -gt 0 ]; then
        kubectl delete namespace "$namespace"
    fi
}

initialize_namespace () {
    check_skip

    namespace="$1"
    
    if [ -z "$SKIP_CHECK_CONTEXT" ]; then
        interactive_check_context "$namespace"
    fi
    
    drop_namespace "$namespace"
    wait_for_namespace_deletion "$namespace"
    kubectl create namespace "$namespace"
}

switch_namespace() {
  kubectl get namespace $1 &> /dev/null
  if [ $? -eq 0 ]; then
    kubectl config set-context $(kubectl config current-context) --namespace=$1 > /dev/null
  else
    echo "namespace $1 does not exist"
    exit 1
  fi
}

cluster_ip () {
    IP=$(kubectl get nodes -ojsonpath="{.items[0].status.addresses[?(@.type==\"ExternalIP\")].address}")

    if [ -z "$IP" ]; then
        IP=$(kubectl cluster-info | fgrep master | python -c 'import sys; print(sys.stdin.readlines()[0].split()[5].split(":")[1].lstrip("/"))')
    fi

    echo "$IP"
}

service_port() {
    namespace=${2:-default}
    instance=${3:-0}

    kubectl get services -n "$namespace" "$1" -ojsonpath="{.spec.ports[$instance].nodePort}"
}

demotest_pod() {
    namespace=${1:-default}

    kubectl get pods -n "$namespace" -l run=demotest -o 'jsonpath={.items[0].metadata.name}'
}

wait_for_namespace_deletion() {
    namespace=${1:-default}
    attempts=100
    running=

    echo "Waiting for namespace deletion"

    while [ $attempts -gt 0 ]; do
        # XXX This " || : " BS at the end is because grep -c -v returns an exit code of
        # 1 when it finds nothing. Stupid grep.
        pending=$(kubectl describe namespaces ${namespace} | grep '^Status:' | grep -c -v Active || :)

        if [ $pending -eq 0 ]; then
            printf "Namespace $namespace cleared.              \n"
            running=YES
            break
        fi

        printf "try %02d: %d being cleared${LINE_END}" $attempts $pending
        attempts=$(( $attempts - 1 ))
        sleep 2
    done

    if [ -z "$running" ]; then
        echo "Namespace $namespace not cleared?" >&2
        exit 1
    fi
}

wait_for_pods () {
    namespace=${1:-default}
    attempts=100
    running=

    while [ $attempts -gt 0 ]; do
        # XXX This " || : " BS at the end is because grep -c -v returns an exit code of
        # 1 when it finds nothing. Stupid grep.
        pending=$(kubectl --namespace ${namespace} describe pods | grep '^Status:' | grep -c -v Running || :)

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
    namespace=${2:-default}
    extra_args=${3}
    attempts=100
    ready=

    while [ $attempts -gt 0 ]; do
        command="curl ${extra_args} -k ${baseurl}/ambassador/v0/check_ready 2>&1 | grep -c 'readiness check OK'"
        OK=$(eval ${command})

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
        echo
        echo "pods running in namespace $namespace -"
        kubectl get pods -n ${namespace} >&2
        echo
        echo "logs from ambassador's pod in namespace $namespace -"
        kubectl logs -n ${namespace} $(ambassador_pod ${namespace}) -c ambassador >&2
        echo
        exit 1
    fi
}

wait_for_zipkin_running () {
    baseurl=${1}
    attempts=100
    ready=

    while [ $attempts -gt 0 ]; do
        OK=$(curl -k -s $baseurl/health | egrep -c '"UP"')

        if [ $OK -gt 0 ]; then
            printf "zipkin ready              \n"
            ready=YES
            break
        fi

        printf "try %02d: zipkin not ready${LINE_END}" $attempts
        attempts=$(( $attempts - 1 ))
        sleep 5
    done

    if [ -z "$ready" ]; then
        echo 'zipkin not yet ready?' >&2
        exit 1
    fi
}

wait_for_extauth_running () {
    baseurl=${1}
    attempts=100
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
    attempts=100
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
    attempts=100
    routed=

    while [ $attempts -gt 0 ]; do
        if checkweights.py "$@"; then
            printf "weights correct              \n"
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
    extra_args=${4}

    rc=1

    command="curl -k -s ${extra_args} ${baseurl}/ambassador/v0/diag/?json=true | jget.py /routes > check-$index.json 2> /dev/null"
    routes=

    attempts=100
    while [ ${attempts} -gt 0 ]; do
        echo "Try $attempts at getting routes from ambassador's diagnostics"

        eval ${command}
        if [ $? -ne 0 ]; then
            echo "error getting routes from diagnostics"
        else
            routes=FOUND
            break
        fi

        attempts=$(( $attempts - 1 ))
        sleep 2
    done

    if [ -z "${routes}" ]; then
        echo "Could not get routes from $baseurl"
        exit 1
    fi

    if ! cmp -s check-$index.json diag-$index.json; then
        echo "check_diag $index: mismatch for $desc"

        if [ -r ${ROOT}/diag-diff.sh ]; then
            if sh ${ROOT}/diag-diff.sh $index; then
                sh ${ROOT}/diag-fix.sh $index
                rc=0
            fi
        else
            diff -u check-$index.json diag-$index.json
        fi
    else
        echo "check_diag $index: OK"
        rc=0
    fi

    return $rc
}

check_listeners () {
    baseurl=$1
    index=$2
    desc=$3

    sleep 20

    rc=1

    curl -k -s ${baseurl}/ambassador/v0/diag/?json=true | jget.py /listeners > check-l-$index.json

    if ! cmp -s check-l-$index.json listeners-$index.json; then
        echo "check_listeners $index: mismatch for $desc"

        if [ -r ${ROOT}/diag-diff.sh ]; then
            if sh ${ROOT}/diag-diff.sh listeners-$index.json check-l-$index.json; then
                sh ${ROOT}/diag-fix.sh listeners-$index.json check-l-$index.json
                rc=0
            fi
        else
            diff -u listeners-$index.json check-l-$index.json
        fi
    else
        echo "check_listeners $index: OK"
        rc=0
    fi

    return $rc
}

istio_running () {
    kubectl get service istio-mixer >/dev/null 2>&1
}

ambassador_pod () {
    namespace=${1:-default}
    apod=$(kubectl get pod -l service=ambassador -n "$namespace" -o jsonpath='{.items[0].metadata.name}')

    if [ $? -ne 0 ]; then
        echo "Could not find Ambassador pod" >&2
        exit 1
    fi

    echo $apod
}

kubectl_context () {
    kubectl config current-context
}

get_http_code() {
    url=$1
    extra_args=$2

    command="curl $extra_args -w %{http_code} -s -o /dev/null $url"
    echo $(eval ${command})
}

check_http_code() {
    url=$1
    extra_args=$2
    expected=$3

    code=$(get_http_code ${url} ${extra_args})
    echo "URL ${url} returned ${code}, expected ${expected}"
    if [ ${code} -ne ${expected} ]; then
        exit 1
    fi
}

get_redirect_url() {
    url=$1
    extra_args=$2

    command="curl $extra_args -w %{redirect_url} -s -o /dev/null $url"
    echo $(eval ${command})
}

interactive_check_context () {
    CONTEXT=$(kubectl_context)
    namespace="$1"

    if [ -n "$namespace" ]; then
        prompt="You are about to delete everything in namespace $namespace of context $CONTEXT."
    else
        prompt="You are about to delete everything in context $CONTEXT."
    fi

    while true; do
        read -p "${prompt} Proceed? [yes, no] " yn
        case $yn in
            [Yy]* ) return;;
            [Nn]* ) exit;;
            * ) echo "Please answer yes or no.";;
        esac
    done
}

get_http_code() {
    url=$1
    extra_args=$2

    command="curl $extra_args -w %{http_code} -s -o /dev/null $url"
    echo $(eval ${command})
}

get_redirect_url() {
    url=$1
    extra_args=$2

    command="curl $extra_args -w %{redirect_url} -s -o /dev/null $url"
    echo $(eval ${command})
}

check_ambassador_diff() {
    pod=$1
    container_file=$2
    local_file=$3

    container_content=$(kubectl exec ${pod} -c ambassador cat ${container_file})
    diff <${container_content} ${local_file}
    exit_code=$?
    if [ ${exit_code} -ne 0 ]; then
        echo "ambassador:${container_file} and ${local_file} do not match"
        exit 1
    else
        echo "ambassador:${container_file} and ${local_file} match"
    fi
}

check_CN() {
    url=$1
    CN=$2

    # grepping on something like
    # `openssl s_client -connect <IP>:<PORT> | openssl x509 -noout -subject`
    # might be cleaner if this does not look ideal.
    output=$(curl -o /dev/null -s -v -k ${url} 2>&1)
    echo "${output}" | grep CN=${CN}
    exit_code=$?
    if [ ${exit_code} -ne 0 ]; then
        echo "SSL certificate at ${url} does not have CN=${CN}, got the following output instead -"
        echo "${output}" | grep CN
        exit 1
    else
        echo "SSL certificate at ${url} has CN=${CN}"
    fi
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
