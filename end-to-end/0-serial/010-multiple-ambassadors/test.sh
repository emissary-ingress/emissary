#!/bin/bash

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

set -e -o pipefail

HERE=$(cd $(dirname $0); pwd)

cd "$HERE"

ROOT=$(cd ../..; pwd)
PATH="${ROOT}:${PATH}"

source ${ROOT}/utils.sh

initialize_cluster

kubectl cluster-info

python ${ROOT}/yfix.py ${ROOT}/fixes/test-dep.yfix \
    ${ROOT}/ambassador-deployment.yaml \
    k8s/ambassador-deployment-1.yaml \
    test-010-1 \
    ambassador-1

python ${ROOT}/yfix.py ${ROOT}/fixes/test-dep.yfix \
    ${ROOT}/ambassador-deployment.yaml \
    k8s/ambassador-deployment-2.yaml \
    test-010-2 \
    ambassador-2

kubectl create namespace test-010-1
kubectl create namespace test-010-2
kubectl create namespace test-010-svc
kubectl apply -f k8s/rbac.yaml
kubectl apply -f k8s/ambassador-1.yaml
kubectl apply -f k8s/ambassador-2.yaml
kubectl apply -f k8s/ambassador-deployment-1.yaml
kubectl apply -f k8s/ambassador-deployment-2.yaml

kubectl run demotest --image=dwflynn/demotest:0.0.1 -- /bin/sh -c "sleep 3600"

set +e +o pipefail

wait_for_pods

CLUSTER=$(cluster_ip)
APORT1=$(service_port ambassador test-010-1)
APORT2=$(service_port ambassador test-010-2)
DEMOTEST_POD=$(demotest_pod)

BASEURL1="http://${CLUSTER}:${APORT1}"
BASEURL2="http://${CLUSTER}:${APORT2}"

echo "Base 1 URL $BASEURL1"
echo "Diag 1 URL $BASEURL1/ambassador/v0/diag/"
echo "Base 2 URL $BASEURL2"
echo "Diag 2 URL $BASEURL2/ambassador/v0/diag/"

wait_for_ready "$BASEURL1"
wait_for_ready "$BASEURL2"

if ! check_diag "$BASEURL1" 1-1 "No annotated services"; then
    exit 1
fi

if ! check_diag "$BASEURL2" 1-2 "No annotated services"; then
    exit 1
fi

kubectl apply -f k8s/demo-1.yaml
kubectl apply -f k8s/demo-2.yaml

wait_for_pods

wait_for_demo_weights "$BASEURL1" 100
wait_for_demo_weights "$BASEURL2" 100


if ! check_diag "$BASEURL1" 2-1 "demo annotated"; then
    exit 1
fi

if ! check_diag "$BASEURL2" 2-2 "demo annotated"; then
    exit 1
fi

if ! kubectl exec -i $DEMOTEST_POD -- python3 demotest.py "$BASEURL1" /dev/fd/0 < demo-1.yaml; then
    exit 1
fi

if ! kubectl exec -i $DEMOTEST_POD -- python3 demotest.py "$BASEURL2" /dev/fd/0 < demo-2.yaml; then
    exit 1
fi

# kubernaut discard
