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

NAMESPACE="003-headers-and-host"

cd $(dirname $0)
ROOT=$(cd ../..; pwd)
source ${ROOT}/utils.sh
bootstrap --cleanup ${NAMESPACE} ${ROOT}

python ${ROOT}/yfix.py ${ROOT}/fixes/ambassador-id.yfix \
    ${ROOT}/ambassador-deployment.yaml \
    k8s/ambassador-deployment.yaml \
    ${NAMESPACE} \
    ${NAMESPACE}

kubectl apply -f k8s/rbac.yaml
kubectl apply -f k8s/ambassador.yaml
kubectl apply -f k8s/cors.yaml
kubectl apply -f k8s/demo1.yaml
kubectl apply -f k8s/demo2.yaml
kubectl apply -f k8s/ambassador-deployment.yaml
kubectl run demotest --image=dwflynn/demotest:0.0.1 -- /bin/sh -c "sleep 3600"

set +e +o pipefail

wait_for_pods ${NAMESPACE}

CLUSTER=$(cluster_ip)
APORT=$(service_port ambassador ${NAMESPACE})
DEMOTEST_POD=$(demotest_pod ${NAMESPACE})

BASEURL="http://${CLUSTER}:${APORT}"

echo "Base URL $BASEURL"
echo "Diag URL $BASEURL/ambassador/v0/diag/"

wait_for_ready "$BASEURL" ${NAMESPACE}

if ! check_diag "$BASEURL" 1 "No canary active"; then
    exit 1
fi

if ! kubectl exec -i $DEMOTEST_POD -- python3 demotest.py "$BASEURL" /dev/fd/0 < cors.yaml; then
    exit 1
fi

if ! kubectl exec -i $DEMOTEST_POD -- python3 demotest.py "$BASEURL" /dev/fd/0 < demo-1.yaml; then
    exit 1
fi

kubectl apply -f k8s/canary-50.yaml
wait_for_pods ${NAMESPACE}
wait_for_demo_weights "$BASEURL" x-demo-mode=canary 50 50

if ! check_diag "$BASEURL" 2 "Canary 50/50"; then
    exit 1
fi

sleep 5

if ! kubectl exec -i $DEMOTEST_POD -- python3 demotest.py "$BASEURL" /dev/fd/0 < demo-2.yaml; then
    exit 1
fi

kubectl apply -f k8s/canary-100.yaml
wait_for_pods ${NAMESPACE}
wait_for_demo_weights "$BASEURL" x-demo-mode=canary 100

if ! check_diag "$BASEURL" 3 "Canary 100"; then
    exit 1
fi

sleep 5

if ! kubectl exec -i "$DEMOTEST_POD" -- python3 demotest.py "$BASEURL" /dev/fd/0 < demo-3.yaml; then
    exit 1
fi

if [ -n "$CLEAN_ON_SUCCESS" ]; then
    drop_namespace ${NAMESPACE}
fi