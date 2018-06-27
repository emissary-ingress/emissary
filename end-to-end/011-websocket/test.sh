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

ROOT=$(cd ..; pwd)
PATH="${ROOT}:${PATH}"

source ${ROOT}/utils.sh

initialize_namespace "ambassador-sbx0"

# Make sure cluster-wide RBAC is set up.
kubectl apply -f ${ROOT}/rbac.yaml

kubectl cluster-info

# Make sure we have a forge.yaml (.gitignore stops us from
# checking it in, which is usually a good thing).
cp forge.yaml.src forge.yaml

# Get stuff deployed.
cd ambassador
$ROOT/forge --profile=sandbox0 deploy
cd ../web-basic
$ROOT/forge --profile=sandbox0 deploy
cd ..

set +e +o pipefail

wait_for_pods ambassador-sbx0

CLUSTER=$(cluster_ip)
APORT=$(service_port ambassador ambassador-sbx0)

BASEURL="http://${CLUSTER}:${APORT}"

echo "Base URL $BASEURL"
echo "Diag URL $BASEURL/ambassador/v0/diag/"

wait_for_ready "$BASEURL"

if ! check_diag "$BASEURL" 1 "No annotated services"; then
    exit 1
fi

python web-basic/wscat.py ws://${CLUSTER}:${APORT}/ws

# kubernaut discard
