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

NAMESPACE="012-xfp-redirect"

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

BASEURL="http://${CLUSTER}:${APORT}"

echo "Base URL $BASEURL"
echo "Diag URL $BASEURL/ambassador/v0/diag/"

extra_args="-H 'X-FORWARDED-PROTO: https'"

wait_for_ready "$BASEURL" ${NAMESPACE} "$extra_args"

if ! check_diag "$BASEURL" 1 "QOTM present" "$extra_args"; then
    exit 1
fi

# Making a normal HTTP request to /qotm/ should result in a redirect to https://
code=$(get_http_code "$BASEURL/qotm/")
if [ $code -ne 301 ]; then
    echo "Expected 301 HTTP code, but got $code"
    exit 1
fi
echo "Get $code for non XFP HTTP request to $BASEURL/qotm/"
redirect_url=$(get_redirect_url "$BASEURL/qotm/")
if [ ${redirect_url} != "https://${CLUSTER}:${APORT}/qotm/" ]; then
    echo "Expected redirect URL to be "https://${CLUSTER}:${APORT}/qotm/", but got $redirect_url"
    exit 1
fi
echo "Ambassador redirected to $redirect_url for non XFP HTTP request to $BASEURL/qotm/"

# Making an HTTP request with X-FORWARDED-PROTO header set to 'https' should return a 200 without any redirects
code=$(get_http_code "$BASEURL/qotm/" "$extra_args")
if [ $code -ne 200 ]; then
    echo "Expected 200 HTTP code since 'X-FORWARDED-PROTO: https' header was set, but got $code"
    exit 1
fi
echo "Got $code for HTTP request with 'X-FORWARDED-PROTO: https' to URL $BASEURL/qotm/"

# Making an HTTP request with X-FORWARDED-PROTO header set to 'http' should still return a 301 redirect
code=$(get_http_code "$BASEURL/qotm/" "-H 'X-FORWARDED-PROTO: http'")
if [ $code -ne 301 ]; then
    echo "Expected 301 HTTP code since 'X-FORWARDED-PROTO: http' header was set, but got $code"
    exit 1
fi
echo "Got $code for HTTP request with 'X-FORWARDED-PROTO: http' set to URL $BASEURL/qotm/"

redirect_url=$(get_redirect_url "$BASEURL/qotm/" "-H 'X-FORWARDED-PROTO: http'")
if [ ${redirect_url} != "https://${CLUSTER}:${APORT}/qotm/" ]; then
    echo "Expected redirect URL to be "https://${CLUSTER}:${APORT}/qotm/", but got $redirect_url"
    exit 1
fi
echo "Ambassador redirected to $redirect_url for HTTP request with 'X-FORWARDED-PROTO: http' to $BASEURL/qotm/"

if [ -n "$CLEAN_ON_SUCCESS" ]; then
    drop_namespace ${NAMESPACE}
fi
