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
kubectl apply -f k8s/rate-limit.yaml

set +e +o pipefail

wait_for_pods

CLUSTER=$(cluster_ip)
APORT=$(service_port ambassador)

BASEURL="http://${CLUSTER}:${APORT}"

echo "Base URL $BASEURL"
echo "Diag URL $BASEURL/ambassador/v0/diag/"

wait_for_ready "$BASEURL"
wait_for_pods

if ! python rate-limit-test.py $CLUSTER:$APORT; then
    exit 1
fi

# kubernaut discard
