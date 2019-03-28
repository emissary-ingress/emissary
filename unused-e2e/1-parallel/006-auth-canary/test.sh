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

NAMESPACE="006-auth-canary"

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
kubectl apply -f k8s/ambassador-deployment.yaml
kubectl apply -f k8s/qotm.yaml
kubectl apply -f k8s/auth-1.yaml

set +e +o pipefail

wait_for_pods ${NAMESPACE}

CLUSTER=$(cluster_ip)
APORT=$(service_port ambassador ${NAMESPACE})

BASEURL="http://${CLUSTER}:${APORT}"

echo "Base URL $BASEURL"
echo "Diag URL $BASEURL/ambassador/v0/diag/"

wait_for_ready "$BASEURL" ${NAMESPACE}

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

wait_for_pods ${NAMESPACE}

wait_for_extauth_enabled "$BASEURL"
sleep 20 # Not sure why this is sometimes relevant.

if ! check_diag "$BASEURL" 3 "Auth 1 and 2"; then
    exit 1
fi

if ! python auth-test.py $CLUSTER:$APORT "1.0.0:50" "2.0.0:50"; then
    exit 1
fi

kubectl delete service auth-1
kubectl delete deployment auth-1

# This works because it'll wait for "terminating" to go away too.
wait_for_pods ${NAMESPACE}

wait_for_extauth_enabled "$BASEURL"
sleep 5 # Not sure why this is sometimes relevant.

if ! check_diag "$BASEURL" 4 "Auth 2 only"; then
    exit 1
fi

if ! python auth-test.py $CLUSTER:$APORT "2.0.0:100"; then
    exit 1
fi

if [ -n "$CLEAN_ON_SUCCESS" ]; then
    drop_namespace ${NAMESPACE}
fi
