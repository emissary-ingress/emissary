#!/usr/bin/env bash
set -o errexit
set -o nounset

KUBECTL_VERSION=1.10.2

curl -LO https://storage.googleapis.com/kubernetes-release/release/v${KUBECTL_VERSION}/bin/linux/amd64/kubectl
chmod +x kubectl
mv kubectl ~/bin/kubectl

pip install -q -r dev-requirements.txt
pip install -q -r ambassador/requirements.txt
npm install gitbook-cli netlify-cli
