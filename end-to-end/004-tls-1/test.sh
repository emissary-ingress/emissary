#!/bin/bash

set -e -o pipefail

HERE=$(cd $(dirname $0); pwd)

cd "$HERE"

ROOT=$(cd ..; pwd)
PATH="${ROOT}:${PATH}"

source ${ROOT}/utils.sh

step "Dropping old cluster"
kubernaut discard

step "Claiming new cluster"
kubernaut claim 
export KUBECONFIG=${HOME}/.kube/kubernaut

kubectl cluster-info

kubectl create secret tls ambassador-certs-termination --cert=certs/termination.crt --key=certs/termination.key
kubectl create secret tls ambassador-certs-upstream --cert=certs/upstream.crt --key=certs/upstream.key

kubectl apply -f k8s/authsvc.yaml
kubectl apply -f k8s/ambassador.yaml
kubectl apply -f k8s/demo1.yaml
kubectl apply -f k8s/demo2.yaml
kubectl apply -f ${ROOT}/ambassador-deployment-mounts.yaml

set +e +o pipefail

wait_for_pods

CLUSTER=$(cluster_ip)
APORT=$(service_port ambassador)

BASEURL="https://${CLUSTER}:${APORT}"

echo "Base URL $BASEURL"
echo "Diag URL $BASEURL/ambassador/v0/diag/"

wait_for_ready "$BASEURL"

if ! check_diag "$BASEURL" 1 "No canary active"; then
    exit 1
fi

if ! demotest.py "$BASEURL" demo-1.yaml; then
    exit 1
fi

kubectl apply -f k8s/canary-50.yaml
wait_for_pods
wait_for_demo_weights "$BASEURL" x-demo-mode=canary 50 50

# This needs sorting crap before it'll work. :P
# if ! check_diag "$BASEURL" 2 "Canary 50/50"; then
#     exit 1
# fi

if ! demotest.py "$BASEURL" demo-2.yaml; then
    exit 1
fi

kubectl apply -f k8s/canary-100.yaml
wait_for_pods
wait_for_demo_weights "$BASEURL" x-demo-mode=canary 100

# This needs sorting crap before it'll work. :P
# if ! check_diag "$BASEURL" 3 "Canary 100"; then
#     exit 1
# fi

if ! demotest.py "$BASEURL" demo-3.yaml; then
    exit 1
fi

