#!/usr/bin/env bash
set -o errexit
set -o nounset
set -o pipefail

KUBECTL_VERSION=${KUBECTL_VERSION:?KUBECTL_VERSION is not set}

curl -L --output ~/bin/kubectl \
     https://storage.googleapis.com/kubernetes-release/release/v${KUBECTL_VERSION}/bin/linux/amd64/kubectl

chmod +x ~/bin/kubectl

kubectl version --client
