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

kubectl apply -f k8s/ambassador.yaml
kubectl apply -f ${ROOT}/ambassador-deployment.yaml
kubectl apply -f k8s/qotm.yaml

set +e +o pipefail

wait_for_pods

CLUSTER=$(cluster_ip)
APORT=$(service_port ambassador)
APOD=$(ambassador_pod)

BASEURL="http://${CLUSTER}:${APORT}"

echo "Base URL $BASEURL"

wait_for_ready "$BASEURL"

check_diag () {
    index=$1
    desc=$2

    rc=1

    kubectl exec "$APOD" -c ambassador -- sh -c 'curl -k -s localhost:8877/ambassador/v0/diag/?json=true' | jget.py /routes > check-$index.json

    if ! cmp -s check-$index.json diag-$index.json; then
        echo "check_diag $index: mismatch for $desc"

        if diag-diff.sh $index; then
            diag-fix.sh $index
            rc=0
        fi
    else
        echo "check_diag $index: OK"
        rc=0
    fi

    return $rc
}

if ! check_diag 1 "No annotated services"; then
    exit 1
fi

diag_status=$(curl -s -o /dev/null -w "%{http_code}" "${BASEURL}/ambassador/v0/diag/?json=true")

if [ "$diag_status" == 404 ]; then
    echo "External diag access prevented"
else
    echo "External diag allowed? $diag_status" >&2
    exit 1
fi

if ! qtest.py $CLUSTER:$APORT test-1.yaml; then
    exit 1
fi

# kubernaut discard
