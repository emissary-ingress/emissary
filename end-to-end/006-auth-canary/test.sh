#!/bin/bash

set -e -o pipefail

HERE=$(cd $(dirname $0); pwd)

cd "$HERE"

ROOT=$(cd ..; pwd)
PATH="${ROOT}:${PATH}"

source ${ROOT}/utils.sh

initialize_cluster

kubectl cluster-info

kubectl apply -f k8s/ambassador.yaml
kubectl apply -f ${ROOT}/ambassador-deployment.yaml
kubectl apply -f k8s/qotm.yaml
kubectl apply -f k8s/auth-1.yaml

set +e +o pipefail

wait_for_pods

CLUSTER=$(cluster_ip)
APORT=$(service_port ambassador)

BASEURL="http://${CLUSTER}:${APORT}"

echo "Base URL $BASEURL"
echo "Diag URL $BASEURL/ambassador/v0/diag/"

wait_for_ready "$BASEURL"

if ! check_diag "$BASEURL" 1 "No auth"; then
    exit 1
fi

if ! python auth-test.py $CLUSTER:$APORT "none:100"; then
    exit 1
fi

kubectl apply -f k8s/auth-1-enable.yaml

wait_for_extauth_enabled "$BASEURL"
sleep 5 # Not sure why this is sometimes relevant.

if ! check_diag "$BASEURL" 2 "Auth 1"; then
    exit 1
fi

if ! python auth-test.py $CLUSTER:$APORT "1.0.0:100"; then
    exit 1
fi

kubectl apply -f k8s/auth-2.yaml

wait_for_pods

wait_for_extauth_enabled "$BASEURL"
sleep 5 # Not sure why this is sometimes relevant.

if ! check_diag "$BASEURL" 3 "Auth 1 and 2"; then
    exit 1
fi

if ! python auth-test.py $CLUSTER:$APORT "1.0.0:50" "2.0.0:50"; then
    exit 1
fi

kubectl delete service auth-1
kubectl delete deployment auth-1

# This works because it'll wait for "terminating" to go away too.
wait_for_pods

wait_for_extauth_enabled "$BASEURL"
sleep 5 # Not sure why this is sometimes relevant.

if ! check_diag "$BASEURL" 4 "Auth 2 only"; then
    exit 1
fi

if ! python auth-test.py $CLUSTER:$APORT "2.0.0:100"; then
    exit 1
fi

# kubernaut discard
