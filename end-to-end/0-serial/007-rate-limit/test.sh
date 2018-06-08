#!/bin/bash

set -e -o pipefail

HERE=$(cd $(dirname $0); pwd)

cd "$HERE"

ROOT=$(cd ../..; pwd)
PATH="${ROOT}:${PATH}"

source ${ROOT}/utils.sh

initialize_cluster

kubectl cluster-info

kubectl apply -f k8s/ambassador.yaml
kubectl apply -f ${ROOT}/ambassador-deployment.yaml
kubectl apply -f k8s/qotm.yaml
kubectl apply -f k8s/rate-limit.yaml

set +e +o pipefail

wait_for_pods

CLUSTER=$(cluster_ip)
APORT=$(service_port ambassador)

BASEURL="http://${CLUSTER}:${APORT}"

echo "Base URL $BASEURL"
echo "Diag URL $BASEURL/ambassador/v0/diag/"

wait_for_ready "$BASEURL"
wait_for_pods

if ! python rate-limit-test.py $CLUSTER:$APORT; then
    exit 1
fi

# kubernaut discard
