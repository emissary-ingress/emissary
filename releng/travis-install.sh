#!/usr/bin/env bash

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

KUBECTL_VERSION=1.10.2
HELM_VERSION=2.9.1
GO_VERSION=1.13

set -o errexit
set -o nounset
set -o xtrace

printf "== Begin: travis-install.sh ==\n"

mkdir -p ~/bin

# Set up for Kubernaut.
base64 -d < kconf.b64 | ( cd ~ ; tar xzf - )

curl -L -o ~/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/v${KUBECTL_VERSION}/bin/linux/amd64/kubectl
chmod +x ~/bin/kubectl

curl -L https://storage.googleapis.com/kubernetes-helm/helm-v${HELM_VERSION}-linux-amd64.tar.gz | tar -x -z -O linux-amd64/helm > ~/bin/helm
chmod +x ~/bin/helm
helm init --client-only # Initialize helm for indexing use

KUBERNAUT_VERSION=2018.10.24-d46c1f1
KUBERNAUT=~/bin/kubernaut
curl -o ${KUBERNAUT} http://releases.datawire.io/kubernaut/${KUBERNAUT_VERSION}/linux/amd64/kubernaut
chmod +x ${KUBERNAUT}

# Configure kubernaut
base64 -d < kconf.b64 | ( cd ~ ; tar xzf - )

# Grab a kubernaut cluster
CLAIM_NAME=kat-${USER}-$(uuidgen)
DEV_KUBECONFIG=~/.kube/${CLAIM_NAME}.yaml
echo $CLAIM_NAME > ~/kubernaut-claim.txt

kubernaut claims delete ${CLAIM_NAME}
kubernaut claims create --name ${CLAIM_NAME} --cluster-group main
kubectl --kubeconfig ${DEV_KUBECONFIG} -n default get service kubernetes

# Once the cluster is live, get the ephemeral Docker registry going.
printf "Starting local Docker registry in Kubernetes\n"

kubectl --kubeconfig ${DEV_KUBECONFIG} apply -f releng/docker-registry.yaml

while true; do
	reg_pod=$(kubectl --kubeconfig ${DEV_KUBECONFIG} get pods -n docker-registry -ojsonpath='{.items[0].status.containerStatuses[0].state.running}')

	if [ -z "$reg_pod" ]; then
		printf "...waiting for registry pod\n"
		sleep 1
	else
		printf "...registry pod ready\n"
		break
	fi
done

# Start the port forwarder running in the background.
kubectl --kubeconfig ${DEV_KUBECONFIG} port-forward --namespace=docker-registry deployment/registry 31000:5000 > /tmp/port-forward-log &

while true; do
	if ! curl -i http://localhost:31000/ 2>/dev/null; then
		printf "...waiting for port forwarding\n"
		sleep 1
	else
		printf "...port forwarding ready\n"
		break
	fi
done

printf "== End:   travis-install.sh ==\n"
