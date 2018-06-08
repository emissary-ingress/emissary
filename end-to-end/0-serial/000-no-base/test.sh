#!/bin/bash

set -e -o pipefail

HERE=$(cd $(dirname $0); pwd)

cd "$HERE"

ROOT=$(cd ..; pwd)
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
