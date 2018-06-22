#!/bin/bash

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
