#!/usr/bin/env bash

set -e
set -u

openssl aes-256-cbc -K $encrypted_3c8a53ca0ead_key -iv $encrypted_3c8a53ca0ead_iv -in key-file.json.enc -out key-file.json -d

# gcloud
gcloud version || true
if [ ! -d "$HOME/google-cloud-sdk/bin" ]; then rm -rf $HOME/google-cloud-sdk; curl https://sdk.cloud.google.com | bash; fi
source /home/travis/google-cloud-sdk/path.bash.inc
gcloud version
gcloud auth activate-service-account $K8S_ACCOUNT_NAME --key-file=./key-file.json
gcloud --quiet config set container/use_client_certificate False
gcloud --quiet config set project $K8S_PROJECT
gcloud --quiet config set container/cluster $K8S_CLUSTER
gcloud --quiet config set compute/zone $K8S_ZONE
gcloud --quiet container clusters get-credentials $K8S_CLUSTER --zone=$K8S_ZONE
gcloud --quiet components install kubectl

# kubectl
kubectl config current-context
if [[ -z $(kubectl get clusterrolebinding --field-selector metadata.name=cluster-admin) ]]; then
  kubectl create clusterrolebinding cluster-admin-binding --clusterrole cluster-admin --user=$K8S_ACCOUNT_NAME --namespace=datawire
fi

