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

NAMESPACE="004-tls-1"

cd $(dirname $0)
ROOT=$(cd ../..; pwd)
source ${ROOT}/utils.sh
bootstrap --cleanup ${NAMESPACE} ${ROOT}

python ${ROOT}/yfix.py ${ROOT}/fixes/ambassador-id.yfix \
    ${ROOT}/ambassador-deployment-mounts.yaml \
    k8s/ambassador-deployment.yaml \
    ${NAMESPACE} \
    ${NAMESPACE}

kubectl apply -f k8s/rbac.yaml

kubectl create secret tls ambassador-certs --cert=certs/termination.crt --key=certs/termination.key
kubectl create secret tls ambassador-certs-upstream --cert=certs/upstream.crt --key=certs/upstream.key

kubectl apply -f k8s/authsvc.yaml
kubectl apply -f k8s/ambassador.yaml
kubectl apply -f k8s/demo1.yaml
kubectl apply -f k8s/demo2.yaml
kubectl apply -f k8s/ambassador-deployment.yaml
kubectl run demotest --image=dwflynn/demotest:0.0.1 -- /bin/sh -c "sleep 3600"

set +e +o pipefail

wait_for_pods ${NAMESPACE}

CLUSTER=$(cluster_ip)
APORT=$(service_port ambassador ${NAMESPACE})
DEMOTEST_POD=$(demotest_pod ${NAMESPACE})

BASEURL="https://${CLUSTER}:${APORT}"
HTTPURL="http://${CLUSTER}:$(service_port ambassador ${NAMESPACE} 1)"

echo "Base URL $BASEURL"
echo "HTTP URL $HTTPURL"
echo "Diag URL $BASEURL/ambassador/v0/diag/"

wait_for_ready "$BASEURL" ${NAMESPACE}

if ! check_diag "$BASEURL" 1 "No canary active"; then
    exit 1
fi

status=$(curl -s --write-out "%{http_code} %{redirect_url}" "$HTTPURL/demo/")
rc=$?

if [ $rc -ne 0 ]; then
    echo "HTTP redirect check failed ($rc): $status" >&2
    exit 1
fi

code=$(echo "$status" | cut -d' ' -f1)
redirect_url=$(echo "$status" | cut -d' ' -f2-)

if [ "$code" != "301" ]; then
    echo "HTTP redirect check got $code instead of 301: $status" >&2
    exit 1
fi

if [ $(echo "$redirect_url" | grep -s -c "^https://${CLUSTER}[/:]") -ne 1 ]; then
    echo "HTTP redirect check goes somewhere weird: $status" >&2
    exit 1
fi

echo "HTTP redirect check passed"

if ! kubectl exec -i $DEMOTEST_POD -- python3 demotest.py "$BASEURL" /dev/fd/0 < demo-1.yaml; then
    exit 1
fi

echo "kubectl apply -f k8s/canary-50.yaml"
kubectl apply -f k8s/canary-50.yaml
wait_for_pods ${NAMESPACE}
wait_for_demo_weights "$BASEURL" x-demo-mode=canary 50 50

# This needs sorting crap before it'll work. :P
# if ! check_diag "$BASEURL" 2 "Canary 50/50"; then
#     exit 1
# fi

if ! kubectl exec -i $DEMOTEST_POD -- python3 demotest.py "$BASEURL" /dev/fd/0 < demo-2.yaml; then
    exit 1
fi

echo "kubectl apply -f k8s/canary-100.yaml"
kubectl apply -f k8s/canary-100.yaml
wait_for_pods ${NAMESPACE}

sleep 10

wait_for_demo_weights "$BASEURL" x-demo-mode=canary 100

# This needs sorting crap before it'll work. :P
# if ! check_diag "$BASEURL" 3 "Canary 100"; then
#     exit 1
# fi

if ! kubectl exec -i $DEMOTEST_POD -- python3 demotest.py "$BASEURL" /dev/fd/0 < demo-3.yaml; then
    exit 1
fi

if [ -n "$CLEAN_ON_SUCCESS" ]; then
    drop_namespace ${NAMESPACE}
fi