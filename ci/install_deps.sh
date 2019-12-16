#!/bin/bash
HELM_VERSION=3.0.1
HELM2_VERSION=2.16.1
KUBECTL_VERSION=1.15.3
KUBERNAUT_VERSION=2018.10.24-d46c1f1


printf "== Begin: travis-install.sh ==\n"

mkdir -p ~/bin
PATH=~/bin:$PATH

## Install kubectl
##
curl -L -o ~/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/v${KUBECTL_VERSION}/bin/linux/amd64/kubectl
chmod +x ~/bin/kubectl

## Install Helm 3
##
curl -L https://storage.googleapis.com/kubernetes-helm/helm-v${HELM_VERSION}-linux-amd64.tar.gz | tar -x -z -O linux-amd64/helm > ~/bin/helm
chmod +x ~/bin/helm

## Install Helm 2
##
curl -L https://storage.googleapis.com/kubernetes-helm/helm-v${HELM2_VERSION}-linux-amd64.tar.gz | tar -x -z -O linux-amd64/helm > ~/bin/helm2
chmod +x ~/bin/helm2

## Install Kubernaut
##
curl -L -o ~/bin/kubernaut http://releases.datawire.io/kubernaut/${KUBERNAUT_VERSION}/linux/amd64/kubernaut
chmod +x ~/bin/kubernaut

## Get Kubernaut cluster
##
base64 -d < ci/kconf.b64 | ( cd ~ ; tar xzf - )

CLAIM_NAME=ambassador-chart-${USER}-$(uuidgen)
DEV_KUBECONFIG=~/.kube/${CLAIM_NAME}.yaml
echo $CLAIM_NAME > ~/kubernaut-claim.txt
kubernaut claims delete ${CLAIM_NAME}
kubernaut claims create --name ${CLAIM_NAME} --cluster-group main
# Do a quick sanity check on that cluster
kubectl --kubeconfig ${DEV_KUBECONFIG} -n default get service kubernetes

printf "== End:   travis-install.sh ==\n"

