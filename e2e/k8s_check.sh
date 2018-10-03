#!/usr/bin/env bash

set -o pipefail
set -o errexit
set -o nounset

echo "Waiting for authorization service deployment..."
bash -c "typeset -i cnt=60"
until kubectl rollout status deployment/auth0-service -n datawire | grep "successfully rolled out"; do 
  ((cnt=cnt-1)) || exit 1; 
  sleep 1; 
done
	
echo "Waiting for ambassador deployment..."
cnt=60
until kubectl rollout status deployment/ambassador -n datawire | grep "successfully rolled out"; do 
  ((cnt=cnt-1)) || exit 1; 
  sleep 1; 
done

SERVICE_IMAGE=$(kubectl get deployment auth0-service -n datawire -o=jsonpath='{$$.spec.template.spec.containers[:1].image}')
MANIFEST_IMAGE=$(cat k8s_build/DEV/authorization-srv.yaml | grep ${SERVICE_IMAGE})
if [ -z "$MANIFEST_IMAGE" ]; then echo "INCORRECT IMAGE"; exit 1; fi
