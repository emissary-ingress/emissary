#!/usr/bin/env bash
set -o errexit
set -o nounset
set -o pipefail

cd infra/loadtest-cluster
terraform plan -out tfplan
terraform apply tfplan
cd ../..

make deploy KUBECONFIG=infra/loadtest-cluster/loadtest.kubeconfig K8S_DIRS=k8s-sidecar_phil
