# Notes

1. `make cluster` and wait... Kubernaut does not work for this so use GKE (note below).
2. Once you have the cluster setup RBAC:
    
    `kubectl apply -f rbac.tiller.yaml`
    
3. Clone Hashicorp's Consul-Helm stuff. Use `master` I wasn't able to get the latest tag to work.

    `git clone https://github.com/hashicorp/consul-helm.git`

4. Install Helm locally: https://docs.helm.sh/using_helm/#installing-helm
5. Install Tiller on cluster: `helm init --service-acount tiller`
6. `cd consul-helm`
7. Edit `values.yaml` and find `connectInject.enabled` and change it to `true`.
8. `helm install --name consul ./`

## Reasons Kubernaut Does Not Work

- Consul uses PersistentVolumeClaims for state and for whatever reason the Kubernaut Kubernetes does not seem to be able to handle PVC (it is a bug that needs investigation.). I thought I could resolve it by ensuring a `StorageClass` resource existed, but that did not fix the problem.
