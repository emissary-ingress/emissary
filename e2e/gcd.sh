#!/bin/sh

set -e
set -u

if [ -n $(TRAVIS_TAG) ]; then
  if [ ! -d "$HOME/google-cloud-sdk/bin" ]; then
    rm -rf $HOME/google-cloud-sdk; curl https://sdk.cloud.google.com | bash;  
  fi

  curl -LO https://storage.googleapis.com/kubernetes-release/release/$KUBECTL_VERSION/bin/linux/amd64/kubectl
  chmod +x ./kubectl
  sudo mv $HOME/google-cloud-sdk/bin/kubectl /usr/local/bin/kubectl
 
  source /home/travis/google-cloud-sdk/path.bash.inc;
  gcloud version;
  gcloud auth activate-service-account $K8S_ACCOUNT_NAME --key-file=./key-file.json;
  gcloud --quiet config set container/use_client_certificate False;
  gcloud --quiet config set project $K8S_PROJECT;
  gcloud --quiet config set container/cluster $K8S_CLUSTER;
  gcloud --quiet config set compute/zone $K8S_ZONE;
  gcloud --quiet container clusters get-credentials $K8S_CLUSTER --zone=$K8S_ZONE;

  kubectl config current-context;
  if [[ -z $(kubectl get clusterrolebinding --field-selector metadata.name=cluster-admin) && -n $TRAVIS_TAG ]]; then
    kubectl create clusterrolebinding cluster-admin-binding --clusterrole cluster-admin --user=$K8S_ACCOUNT_NAME --namespace=datawire;
  fi  
fi
