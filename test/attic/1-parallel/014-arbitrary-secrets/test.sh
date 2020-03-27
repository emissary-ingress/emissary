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

NAMESPACE="014-arbitrary-secrets"

cd $(dirname $0)
ROOT=$(cd ../..; pwd)
source ${ROOT}/utils.sh
bootstrap --cleanup ${NAMESPACE} ${ROOT}

kubectl apply -f k8s/rbac.yaml
kubectl apply -f k8s/ambassador.yaml
kubectl apply -f k8s/qotm.yaml

set +e +o pipefail

CLUSTER=$(cluster_ip)
APORT=$(service_port ambassador ${NAMESPACE})

BASEURL="https://${CLUSTER}:${APORT}"

echo "Base URL $BASEURL"
echo "Diag URL $BASEURL/ambassador/v0/diag/"

MOUNT_SECRET="mount-secret"
MOUNT_CA_SECRET="mount-ca-secret"

USER_SECRET="user-secret"
USER_CA_SECRET="user-ca-secret"

python ${ROOT}/yfix.py ${ROOT}/fixes/test-dep.yfix \
    ${ROOT}/ambassador-deployment.yaml \
    k8s/ambassador-deployment.yaml \
    ${NAMESPACE} \
    ${NAMESPACE}

python ${ROOT}/yfix.py ${ROOT}/fixes/mount-secrets.yfix \
    k8s/ambassador-deployment.yaml \
    k8s/ambassador-secrets-deployment.yaml \
    ${MOUNT_SECRET} \
    ${MOUNT_CA_SECRET}

# Test case 1
echo "==== 1 ===="
echo "Input: certs are manually mounted, no other secret is present"
echo "Output: manually mounted certs are used"

kubectl create secret tls ${MOUNT_SECRET} --cert=certs/mount.crt --key=certs/mount.key
kubectl create secret generic ${MOUNT_CA_SECRET} --from-file=tls.crt=certs/mount-ca.crt
echo "Created mount secrets"

kubectl apply -f k8s/ambassador-secrets-deployment.yaml
kubectl apply -f k8s/ambassador-tls-enabled.yaml

wait_for_pods ${NAMESPACE}
wait_for_ready "$BASEURL" ${NAMESPACE}

if ! check_diag "$BASEURL" 1 "QOTM present"; then
    exit 1
fi

POD=$(ambassador_pod ${NAMESPACE})
check_ambassador_diff ${POD} /etc/certs/tls.crt certs/mount.crt
check_ambassador_diff ${POD} /etc/certs/tls.key certs/mount.key
check_ambassador_diff ${POD} /etc/cacert/tls.crt certs/mount-ca.crt

check_CN ${BASEURL}/qotm/ mount.datawire.io
check_http_code ${BASEURL}/qotm/ "-k" 200

# Test case 2
echo "==== 2 ===="
echo "Input: certs are manually mounted, secret ambassador-certs exists"
echo "Output: manually mounted certs are used"

kubectl create secret tls ambassador-certs --cert=certs/ambassador.crt --key=certs/ambassador.key
kubectl create secret generic ambassador-cacert --from-file=tls.crt=certs/ambassador-ca.crt
echo "Created ambassador secrets"

kubectl apply -f k8s/ambassador-secrets-deployment.yaml
kubectl apply -f k8s/ambassador-tls-enabled.yaml

kubectl delete pods -l service=ambassador --force --grace-period=0

wait_for_pods ${NAMESPACE}
wait_for_ready "$BASEURL" ${NAMESPACE}

if ! check_diag "$BASEURL" 1 "QOTM present"; then
    exit 1
fi

POD=$(ambassador_pod ${NAMESPACE})
check_ambassador_diff ${POD} /etc/certs/tls.crt certs/mount.crt
check_ambassador_diff ${POD} /etc/certs/tls.key certs/mount.key
check_ambassador_diff ${POD} /etc/cacert/tls.crt certs/mount-ca.crt

check_CN ${BASEURL}/qotm/ mount.datawire.io
check_http_code ${BASEURL}/qotm/ "-k" 200

# Test case 3
echo "==== 3 ===="
echo "Input: certs are manually mounted, secret ambassador-certs exists, arbitrary-secret is defined in TLS module"
echo "Output: manually mounted certs are used"

kubectl create secret tls ${USER_SECRET} --cert=certs/user.crt --key=certs/user.key
kubectl create secret generic ${USER_CA_SECRET} --from-file=tls.crt=certs/user-ca.crt
echo "Created user-defined secrets"

kubectl apply -f k8s/ambassador-arbitrary-secrets.yaml
kubectl apply -f k8s/ambassador-secrets-deployment.yaml

kubectl delete pods -l service=ambassador --force --grace-period=0

wait_for_pods ${NAMESPACE}
wait_for_ready "$BASEURL" ${NAMESPACE}

if ! check_diag "$BASEURL" 1 "QOTM present"; then
    exit 1
fi

POD=$(ambassador_pod ${NAMESPACE})
check_ambassador_diff ${POD} /etc/certs/tls.crt certs/mount.crt
check_ambassador_diff ${POD} /etc/certs/tls.key certs/mount.key
check_ambassador_diff ${POD} /etc/cacert/tls.crt certs/mount-ca.crt

check_CN ${BASEURL}/qotm/ mount.datawire.io
check_http_code ${BASEURL}/qotm/ "-k" 200

# Test case 4
echo "==== 4 ===="
echo "Input: certs are manually mounted, arbitrary-secret is defined in TLS module"
echo "Output: manually mounted certs are used"

kubectl delete secrets ambassador-certs ambassador-cacert

kubectl apply -f k8s/ambassador-arbitrary-secrets.yaml
kubectl apply -f k8s/ambassador-secrets-deployment.yaml

kubectl delete pods -l service=ambassador --force --grace-period=0

wait_for_pods ${NAMESPACE}
wait_for_ready "$BASEURL" ${NAMESPACE}

if ! check_diag "$BASEURL" 1 "QOTM present"; then
    exit 1
fi

POD=$(ambassador_pod ${NAMESPACE})
check_ambassador_diff ${POD} /etc/certs/tls.crt certs/mount.crt
check_ambassador_diff ${POD} /etc/certs/tls.key certs/mount.key
check_ambassador_diff ${POD} /etc/cacert/tls.crt certs/mount-ca.crt

check_CN ${BASEURL}/qotm/ mount.datawire.io
check_http_code ${BASEURL}/qotm/ "-k" 200

# Test case 5
echo "==== 5 ===="
echo "Input: secret ambassador-certs exists, no other secret is present"
echo "Output: ambassador-certs should be used"

kubectl create secret tls ambassador-certs --cert=certs/ambassador.crt --key=certs/ambassador.key
kubectl create secret generic ambassador-cacert --from-file=tls.crt=certs/ambassador-ca.crt
echo "Created ambassador secrets"

kubectl apply -f k8s/ambassador.yaml
kubectl apply -f k8s/ambassador-deployment.yaml

wait_for_pods ${NAMESPACE}
wait_for_ready "$BASEURL" ${NAMESPACE}

if ! check_diag "$BASEURL" 2 "QOTM present"; then
    exit 1
fi

POD=$(ambassador_pod ${NAMESPACE})
check_ambassador_diff ${POD} /ambassador/certs/tls.crt certs/ambassador.crt
check_ambassador_diff ${POD} /ambassador/certs/tls.key certs/ambassador.key
check_ambassador_diff ${POD} /ambassador/cacert/tls.crt certs/ambassador-ca.crt

check_CN ${BASEURL}/qotm/ ambassador.datawire.io
check_http_code ${BASEURL}/qotm/ "-k" 200

# Test case 6
echo "==== 6 ===="
echo "Input: secret ambassador-certs exists, arbitrary-secret is defined in TLS module"
echo "Output: ambassador-certs should be used"

kubectl apply -f k8s/ambassador-arbitrary-secrets.yaml
kubectl apply -f k8s/ambassador-deployment.yaml

kubectl delete pods -l service=ambassador --force --grace-period=0

wait_for_pods ${NAMESPACE}
wait_for_ready "$BASEURL" ${NAMESPACE}

if ! check_diag "$BASEURL" 1 "QOTM present"; then
    exit 1
fi

POD=$(ambassador_pod ${NAMESPACE})
check_ambassador_diff ${POD} /ambassador/certs/tls.crt certs/ambassador.crt
check_ambassador_diff ${POD} /ambassador/certs/tls.key certs/ambassador.key
check_ambassador_diff ${POD} /ambassador/cacert/tls.crt certs/ambassador-ca.crt

check_CN ${BASEURL}/qotm/ ambassador.datawire.io
check_http_code ${BASEURL}/qotm/ "-k" 200

# Test case 7
echo "==== 7 ===="
echo "Input: arbitrary-secret is defined in TLS module, no other secret is present"
echo "Output: arbitrary-secret should be used"

kubectl delete secrets ambassador-certs ambassador-cacert
kubectl apply -f k8s/ambassador-arbitrary-secrets.yaml
kubectl apply -f k8s/ambassador-deployment.yaml

kubectl delete pods -l service=ambassador --force --grace-period=0

wait_for_pods ${NAMESPACE}
wait_for_ready "$BASEURL" ${NAMESPACE}

if ! check_diag "$BASEURL" 1 "QOTM present"; then
    exit 1
fi

POD=$(ambassador_pod ${NAMESPACE})
check_ambassador_diff ${POD} /ambassador/certs/tls.crt certs/user.crt
check_ambassador_diff ${POD} /ambassador/certs/tls.key certs/user.key
check_ambassador_diff ${POD} /ambassador/cacert/tls.crt certs/user-ca.crt

check_CN ${BASEURL}/qotm/ user.datawire.io
check_http_code ${BASEURL}/qotm/ "-k" 200

if [ -n "$CLEAN_ON_SUCCESS" ]; then
    drop_namespace ${NAMESPACE}
fi
