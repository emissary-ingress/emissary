#!/bin/bash

set -e -o pipefail

HERE=$(cd $(dirname $0); pwd)

cd "$HERE"

ROOT=$(cd ..; pwd)
PATH="${ROOT}:${PATH}"

source ${ROOT}/utils.sh

step "Building images"
docker build -t dwflynn/demo:1.0.0 --build-arg VERSION=1.0.0 demo-service
docker build -t dwflynn/demo:2.0.0 --build-arg VERSION=2.0.0 demo-service
docker push dwflynn/demo:1.0.0
docker push dwflynn/demo:2.0.0

step "Dropping old cluster"
kubernaut discard

step "Claiming new cluster"
kubernaut claim 
export KUBECONFIG=${HOME}/.kube/kubernaut

kubectl cluster-info

kubectl apply -f k8s
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

if ! demotest.py "$BASEURL" demo-1.yaml; then
    exit 1
fi
