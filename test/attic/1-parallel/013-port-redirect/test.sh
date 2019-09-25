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

NAMESPACE="013-port-redirect"

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
kubectl apply -f k8s/qotm.yaml

set +e +o pipefail

wait_for_pods ${NAMESPACE}

CLUSTER=$(cluster_ip)
APORT=$(service_port ambassador ${NAMESPACE})

REDIRECTPORT=$(service_port ambassador ${NAMESPACE} 1)
ADMINPORT=$(service_port ambassador-admin ${NAMESPACE})

BASEURL="http://${CLUSTER}:${APORT}"
ADMINURL="http://${CLUSTER}:${ADMINPORT}"
REDIRECTURL="http://${CLUSTER}:${REDIRECTPORT}"

echo "Base URL $BASEURL"
echo "Diag URL $ADMINURL/ambassador/v0/diag/"

wait_for_ready "$ADMINURL" ${NAMESPACE}

if ! check_diag "$ADMINURL" 1 "QOTM present"; then
    exit 1
fi

# The HTTP request to the the redirecting port should result in a 301
# This is where HTTP requests from the L4 load balancer will be forwarded.
code=$(get_http_code "$REDIRECTURL/qotm/")
if [ $code -ne 301 ]; then
    echo "Expected 301 HTTP code, but got $code"
    exit 1
fi
echo "Got $code for request to $REDIRECTURL/qotm/"
redirect_url=$(get_redirect_url "$REDIRECTURL/qotm/")
if [ ${redirect_url} != "https://${CLUSTER}:${REDIRECTPORT}/qotm/" ]; then
    echo "Expected redirect URL to be "https://${CLUSTER}:${REDIRECTPORT}/qotm/", but got $redirect_url"
    exit 1
fi
echo "Ambassador redirected to $redirect_url for HTTP request to $REDIRECTURL/qotm/"

# HTTP request to regular ambassador's port (443 in this case) should go through without any TLS certs
# This is where the HTTPS requests from the L4 load balancer will be forwarded to after terminating TLS.
code=$(get_http_code "$BASEURL/qotm/")
if [ $code -ne 200 ]; then
    echo "Expected 200 HTTP code, but got $code"
    exit 1
fi
echo "Got $code for request to $BASEURL/qotm/"

if [ -n "$CLEAN_ON_SUCCESS" ]; then
    drop_namespace ${NAMESPACE}
fi
