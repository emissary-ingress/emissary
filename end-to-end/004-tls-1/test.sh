#!/bin/bash

set -e -o pipefail

HERE=$(cd $(dirname $0); pwd)

cd "$HERE"

ROOT=$(cd ..; pwd)
PATH="${ROOT}:${PATH}"

source ${ROOT}/utils.sh

shred_and_reclaim

kubectl cluster-info

kubectl create secret tls ambassador-certs-termination --cert=certs/termination.crt --key=certs/termination.key
kubectl create secret tls ambassador-certs-upstream --cert=certs/upstream.crt --key=certs/upstream.key

kubectl apply -f k8s/authsvc.yaml
kubectl apply -f k8s/ambassador.yaml
kubectl apply -f k8s/demo1.yaml
kubectl apply -f k8s/demo2.yaml
kubectl apply -f ${ROOT}/ambassador-deployment-mounts.yaml
kubectl run demotest --image=dwflynn/demotest:0.0.1 -- /bin/sh -c "sleep 3600"

set +e +o pipefail

wait_for_pods

CLUSTER=$(cluster_ip)
APORT=$(service_port ambassador)
DEMOTEST_POD=$(kubectl get pods  | grep demotest | awk ' { print $1 }')

BASEURL="https://${CLUSTER}:${APORT}"

echo "Base URL $BASEURL"
echo "Diag URL $BASEURL/ambassador/v0/diag/"

wait_for_ready "$BASEURL"

if ! check_diag "$BASEURL" 1 "No canary active"; then
    exit 1
fi

if ! kubectl exec -i $DEMOTEST_POD -- python3 demotest.py "$BASEURL" /dev/fd/0 < demo-1.yaml; then
    exit 1
fi

kubectl apply -f k8s/canary-50.yaml
wait_for_pods
wait_for_demo_weights "$BASEURL" x-demo-mode=canary 50 50

# This needs sorting crap before it'll work. :P
# if ! check_diag "$BASEURL" 2 "Canary 50/50"; then
#     exit 1
# fi

if ! kubectl exec -i $DEMOTEST_POD -- python3 demotest.py "$BASEURL" /dev/fd/0 < demo-2.yaml; then
    exit 1
fi

kubectl apply -f k8s/canary-100.yaml
wait_for_pods
wait_for_demo_weights "$BASEURL" x-demo-mode=canary 100

# This needs sorting crap before it'll work. :P
# if ! check_diag "$BASEURL" 3 "Canary 100"; then
#     exit 1
# fi

if ! kubectl exec -i $DEMOTEST_POD -- python3 demotest.py "$BASEURL" /dev/fd/0 < demo-3.yaml; then
    exit 1
fi

