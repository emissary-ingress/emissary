#!/bin/bash

set -e -o pipefail

HERE=$(cd $(dirname $0); pwd)

cd "$HERE"

ROOT=$(cd ..; pwd)
PATH="${ROOT}:${PATH}"

source ${ROOT}/utils.sh

shred_and_reclaim

kubectl cluster-info

kubectl create cm ambassador-config --from-file k8s/base-config.yaml
kubectl apply -f k8s/ambassador.yaml
kubectl apply -f ${ROOT}/ambassador-deployment.yaml

set +e +o pipefail

wait_for_pods

CLUSTER=$(cluster_ip)
APORT=$(service_port ambassador)

BASEURL="http://${CLUSTER}:${APORT}"

echo "Base URL $BASEURL"
echo "Diag URL $BASEURL/ambassador/v0/diag/"

wait_for_ready "$BASEURL"

if ! check_diag "$BASEURL" 1 "No annotated services"; then
    exit 1
fi

kubectl apply -f k8s/qotm.yaml

wait_for_pods

if ! check_diag "$BASEURL" 2 "QoTM annotated"; then
    exit 1
fi

if ! qtest.py $CLUSTER:$APORT test-1.yaml; then
    exit 1
fi

kubectl apply -f k8s/authsvc.yaml

wait_for_pods

wait_for_extauth_running "$BASEURL"

kubectl apply -f k8s/authenable.yaml

wait_for_extauth_enabled "$BASEURL"

if ! check_diag "$BASEURL" 3 "Auth annotated"; then
    exit 1
fi

if ! qtest.py $CLUSTER:$APORT test-2.yaml; then
    exit 1
fi

# kubernaut discard
