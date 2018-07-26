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

NAMESPACE="009-grpc"

cd $(dirname $0)
ROOT=$(cd ../..; pwd)
source ${ROOT}/utils.sh
bootstrap --cleanup ${NAMESPACE} ${ROOT}

python ${ROOT}/yfix.py ${ROOT}/fixes/test-dep.yfix \
    ${ROOT}/ambassador-deployment.yaml \
    k8s/ambassador-deployment.yaml \
    ${NAMESPACE} \
    ${NAMESPACE}

kubectl apply -f k8s/rbac.yaml
kubectl apply -f k8s/ambassador.yaml
kubectl apply -f k8s/ambassador-deployment.yaml
kubectl apply -f k8s/grpc.yaml
# kubectl run demotest -n ${NAMESPACE} --image=dwflynn/demotest:0.0.1 -- /bin/sh -c "sleep 3600"

set +e +o pipefail

wait_for_pods ${NAMESPACE}

CLUSTER=$(cluster_ip)
APORT=$(service_port ambassador ${NAMESPACE})
# DEMOTEST_POD=$(demotest_pod)

BASEURL="http://${CLUSTER}:${APORT}"

echo "Base URL $BASEURL"
echo "Diag URL $BASEURL/ambassador/v0/diag/"

wait_for_ready "$BASEURL" ${NAMESPACE}

if ! check_diag "$BASEURL" 1 "grpc annotated"; then
    exit 1
fi

check_grpc () {
    number="$1"

    name="Test Name $number"
    count=$(sh grpcurl.sh -plaintext -d "{\"name\": \"$name\"}" ${CLUSTER}:${APORT} helloworld.Greeter/SayHello | \
            jget.py /message | \
            grep -c "Hello, $name" || true)
    echo "$count $number"
}

echo "Starting GRPC calls"

cp /dev/null count.log

iterations=10
for i in $(seq 1 ${iterations}); do
    check_grpc $i >> count.log &
done

wait

failures=$(egrep -c -v '^1 ' count.log)

if [ $failures -gt 0 ]; then
    echo "FAILED"
    cat count.log
    exit 1
else
    echo "OK"

    if [ -n "$CLEAN_ON_SUCCESS" ]; then
        drop_namespace ${NAMESPACE}
    fi

    exit 0
fi
