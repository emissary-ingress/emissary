#!/bin/sh
TRAVIS_TAG=ok

# k8s generated scripts
PROFILE=prod

# This is the namespace you want to use for development in quay,
# probably your quay.io username:
REGISTRY_NAMESPACE=datawire

# This is the external ip address of the ambassador you are using to
# route to the auth service under development:
EXTERNAL_IP=35.194.24.67

# THese are set from your auth0 account:
AUTH_CALLBACK_URL=http://35.194.24.67/callback
AUTH_DOMAIN=rus123.auth0.com
AUTH_AUDIENCE=https://rus123.auth0.com/api/v2/
AUTH_CLIENT_ID=zTqxrmhGMqZ1J2TjML6QQTsh_kYqJrKv
AUTH_CLIENT_SECRET=auN3fneKuu5HvEFU2wz_swLKBgje3mYbRJj45acXxzQRe_9FsfPuKRLKcDVbNH5H

# Kubernetes
CLOUDSDK_CORE_DISABLE_PROMPTS=1
K8S_PROJECT=datawireio
K8S_CLUSTER=ambassador-oauth-testing
K8S_ZONE=us-central1-a
K8S_ACCOUNT_NAME=ambassador-oauth@datawireio.iam.gserviceaccount.com

if [ -n "$TRAVIS_TAG" ]; then
  openssl aes-256-cbc -K $encrypted_3c8a53ca0ead_key -iv $encrypted_3c8a53ca0ead_iv -in key-file.json.enc -out key-file.json -d
  if [ ! -d "$HOME/google-cloud-sdk/bin" ]; then
    rm -rf $HOME/google-cloud-sdk; curl https://sdk.cloud.google.com | bash;
  fi
  source /home/travis/google-cloud-sdk/path.bash.inc;
  gcloud version;
  gcloud auth activate-service-account $K8S_ACCOUNT_NAME --key-file=./key-file.json;
  gcloud --quiet config set container/use_client_certificate False;
  gcloud --quiet config set project $K8S_PROJECT;
  gcloud --quiet config set container/cluster $K8S_CLUSTER;
  gcloud --quiet config set compute/zone $K8S_ZONE;
  gcloud --quiet container clusters get-credentials $K8S_CLUSTER --zone=$K8S_ZONE;
  gcloud --quiet components install kubectl;
  kubectl config current-context;

  if [[ -z $(kubectl get clusterrolebinding --field-selector metadata.name=cluster-admin) && -n $TRAVIS_TAG ]]; then
    kubectl create clusterrolebinding cluster-admin-binding --clusterrole cluster-admin --user=$K8S_ACCOUNT_NAME --namespace=datawire;
  fi

  make deploy

  echo "INFO: Waiting for authorizarion service deployment..."
  typeset -i cnt=60
  until kubectl rollout status deployment/auth0-service -n datawire | grep "successfully rolled out"; do
    ((cnt=cnt-1)) || exit 1
    sleep 2
  done

  echo "INFO: Waiting for ambassador deployment..."
  typeset -i cnt=60
  until kubectl rollout status deployment/ambassador -n datawire | grep "successfully rolled out"; do
    ((cnt=cnt-1)) || exit 1
    sleep 2
  done
fi