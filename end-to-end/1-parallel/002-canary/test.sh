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

NAMESPACE="002-canary"

cd $(dirname $0)
ROOT=$(cd ../..; pwd)
source ${ROOT}/utils.sh
bootstrap --cleanup ${NAMESPACE} ${ROOT}

python ${ROOT}/yfix.py ${ROOT}/fixes/test-dep.yfix \
    ${ROOT}/ambassador-deployment.yaml \
    k8s/ambassador-deployment.yaml \
    ${NAMESPACE} \
    ${NAMESPACE}

kubectl create namespace ${NAMESPACE}-other
kubectl apply -f k8s/rbac.yaml
kubectl apply -f k8s/ambassador.yaml
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

if ! check_diag "$BASEURL" 1 "No annotated services"; then
    exit 1
fi

kubectl apply -f k8s/demo1.yaml
wait_for_pods ${NAMESPACE}
wait_for_demo_weights "$BASEURL" 100

if ! check_diag "$BASEURL" 2 "demo1 annotated"; then
    exit 1
fi

if ! kubectl exec $DEMOTEST_POD -- python3 demotest.py $BASEURL 0; then
    exit 1
fi

\kubectl apply -f k8s/demo2.yaml
wait_for_pods ${NAMESPACE}
wait_for_demo_weights "$BASEURL" 90 10

# This needs sorting crap before it'll work. :P
# if ! check_diag "$BASEURL" 3 "demo2 annotated"; then
#     exit 1
# fi

if ! kubectl exec $DEMOTEST_POD -- python3 demotest.py $BASEURL 10; then
    exit 1
fi

\kubectl apply -f k8s/demo2-50.yaml
wait_for_pods ${NAMESPACE}
wait_for_demo_weights "$BASEURL" 50 50

# This needs sorting crap before it'll work. :P
# if ! check_diag "$BASEURL" 3 "demo2 annotated"; then
#     exit 1
# fi

if ! kubectl exec $DEMOTEST_POD -- python3 demotest.py $BASEURL 50; then
    exit 1
fi

\kubectl apply -f k8s/demo2-90.yaml
wait_for_pods ${NAMESPACE}
wait_for_demo_weights "$BASEURL" 10 90

# This needs sorting crap before it'll work. :P
# if ! check_diag "$BASEURL" 3 "demo2 annotated"; then
#     exit 1
# fi

if ! kubectl exec $DEMOTEST_POD -- python3 demotest.py $BASEURL 90; then
    exit 1
fi

kubectl delete -f k8s/demo1.yaml
wait_for_pods ${NAMESPACE}
wait_for_demo_weights "$BASEURL" 100

# This needs sorting crap before it'll work. :P
# if ! check_diag "$BASEURL" 3 "demo2 annotated"; then
#     exit 1
# fi

if ! kubectl exec $DEMOTEST_POD -- python3 demotest.py $BASEURL 100; then
    exit 1
fi

if [ -n "$CLEAN_ON_SUCCESS" ]; then
    drop_namespace ${NAMESPACE}
    drop_namespace ${NAMESPACE}-other
fi