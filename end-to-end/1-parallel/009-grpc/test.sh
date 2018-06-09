#!/bin/bash

set -ex -o pipefail

HERE=$(cd $(dirname $0); pwd)

cd "$HERE"

ROOT=$(cd ../..; pwd)
PATH="${ROOT}:${PATH}"

source ${ROOT}/utils.sh

check_rbac

initialize_namespace "009-grpc"

kubectl cluster-info

python ${ROOT}/yfix.py ${ROOT}/fixes/test-dep.yfix \
    ${ROOT}/ambassador-deployment.yaml \
    k8s/ambassador-deployment.yaml \
    009-grpc \
    009-grpc

kubectl apply -f k8s/rbac.yaml
kubectl apply -f k8s/ambassador.yaml
kubectl apply -f k8s/ambassador-deployment.yaml
kubectl apply -f k8s/grpc.yaml
# kubectl run demotest -n 009-grpc --image=dwflynn/demotest:0.0.1 -- /bin/sh -c "sleep 3600"

set +e +o pipefail

wait_for_pods 009-grpc

CLUSTER=$(cluster_ip)
APORT=$(service_port ambassador 009-grpc)
# DEMOTEST_POD=$(demotest_pod)

BASEURL="http://${CLUSTER}:${APORT}"

echo "Base URL $BASEURL"
echo "Diag URL $BASEURL/ambassador/v0/diag/"

wait_for_ready "$BASEURL"

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
    exit 0
fi
