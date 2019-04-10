#!/usr/bin/env bash
set -o errexit
set -o nounset
set -o pipefail
PS4='$ '; set -x

cd infra/loadtest-cluster
terraform init
terraform plan -out tfplan
terraform apply tfplan
cd ../..

make deploy KUBECONFIG=infra/loadtest-cluster/loadtest.kubeconfig K8S_DIRS=k8s-sidecar_phil
