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

NAMESPACE="001-simple"

cd $(dirname $0)
ROOT=$(cd ../..; pwd)
source ${ROOT}/utils.sh
bootstrap --cleanup ${NAMESPACE} ${ROOT}

rm -f adep-tmp.yaml

python ${ROOT}/yfix.py ${ROOT}/fixes/test-dep.yfix \
    ${ROOT}/ambassador-deployment.yaml \
    adep-tmp.yaml \
    ${NAMESPACE} \
    ${NAMESPACE}

python ${ROOT}/yfix.py ${ROOT}/fixes/ambassador-not-root.yfix \
    adep-tmp.yaml \
    adep-tmp-2.yaml

python ${ROOT}/yfix.py ${ROOT}/fixes/enable-statsd.yfix \
    adep-tmp-2.yaml \
    k8s/ambassador-deployment.yaml

rm -f adep-tmp*.yaml

kubectl apply -f k8s/rbac.yaml

kubectl create cm ambassador-config --namespace=${NAMESPACE} --from-file k8s/base-config.yaml

kubectl apply -f k8s/ambassador.yaml
kubectl apply -f k8s/ambassador-deployment.yaml
kubectl apply -f k8s/stats-test.yaml

set +e +o pipefail

wait_for_pods ${NAMESPACE}

CLUSTER=$(cluster_ip)
APORT=$(service_port ambassador ${NAMESPACE})

BASEURL="http://${CLUSTER}:${APORT}"

echo "Base URL $BASEURL"
echo "Diag URL $BASEURL/ambassador/v0/diag/"

wait_for_ready "$BASEURL" ${NAMESPACE}

sleep 10

if ! check_diag "$BASEURL" 1 "No annotated services"; then
    exit 1
fi

kubectl apply -f k8s/qotm.yaml

wait_for_pods ${NAMESPACE}

sleep 10 

if ! check_diag "$BASEURL" 2 "QoTM annotated"; then
    exit 1
fi

sleep 5  # ???

if ! qtest.py $CLUSTER:$APORT test-1.yaml; then
    exit 1
fi

kubectl apply -f k8s/authsvc.yaml

wait_for_pods ${NAMESPACE}

wait_for_extauth_running "$BASEURL"

kubectl apply -f k8s/authenable.yaml

wait_for_extauth_enabled "$BASEURL"

if ! check_diag "$BASEURL" 3 "Auth annotated"; then
    exit 1
fi

sleep 5  # ???

if ! qtest.py $CLUSTER:$APORT test-2.yaml; then
    exit 1
fi

set -e

APOD=$(ambassador_pod ${NAMESPACE})
kubectl exec "$APOD" -n ${NAMESPACE} -c ambassador -- sh -c 'echo "DUMP" | nc -u -w 1 statsd-sink 8125' > stats.json

rqt=$(jget.py envoy.cluster.cluster_qotm.upstream_rq_total < stats.json)

rc=0

if [ $rqt != 18 ]; then
    echo "Upstream RQ total mismatch: wanted 18, got $rqt"
    rc=1
else
    echo "Upstream RQ total stat good"
fi

rq200=$(jget.py envoy.cluster.cluster_qotm.upstream_rq_2xx < stats.json)

if [ $rq200 != 12 ]; then
    echo "Upstream RQ 200 mismatch: wanted 12, got $rq200"
    rc=1
else
    echo "Upstream RQ 200 stat good"
fi

if [ \( $rc -eq 0 \) -a \( -n "$CLEAN_ON_SUCCESS" \) ]; then
    drop_namespace ${NAMESPACE}
fi

exit $rc

# kubernaut discard
