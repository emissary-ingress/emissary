# Setup Notes

1. `make cluster` and wait... Kubernaut does not work for this so use GKE (note below).
2. Once you have the cluster setup RBAC:
    
    `kubectl apply -f rbac.tiller.yaml`
    
3. Clone HashiCorp's Consul-Helm stuff. Use `master` I was not able to get the latest tag to work.

    `git clone https://github.com/hashicorp/consul-helm.git`

4. Install Helm locally: https://docs.helm.sh/using_helm/#installing-helm
5. Install Tiller on cluster: `helm init --service-acount tiller`
6. `cd consul-helm`
7. Edit `values.yaml` and find `connectInject.enabled` and change it to `true`.
8. Edit `values.yaml` and find `client.enabled` and set it to `true'.
9. Edit `values.yaml` and find `client.grpc` and set it to `true`.
8. `helm install --name consul ./`
9. Install Ambassador `kubectl apply -f ambassador/`
10. Install the echo client and server `kubectal apply -f pod.echo-client.yaml pod.echo-server.yaml`
11. Install the mappings `kubectl apply -f mappings.yaml`
12. Ambassador's Envoy will have failed to start. Launch another terminal and `kubectl exec <ambassador-pod-id> -c ambassador -- /bin/sh`
13. `envoy --base-id=1337 -c bootstrap-ads.yaml`

## Testing

1. Get the LoadBalancer "External-IP" via `kubectl get svc ambassador`.
2. `curl -v -k https://<External-IP>/static-server`.

## Reason why Envoy fails to start

Ambassador's Envoy and Connect's Envoy are fighting over shared memory space. Ambassador's Envoy needs to be passed `--base-id` value.

## Reasons Kubernaut Does Not Work

- Consul uses PersistentVolumeClaims for state and for whatever reason the Kubernaut Kubernetes does not seem to be able to handle PVC (it is a bug that needs investigation.). I thought I could resolve it by ensuring a `StorageClass` resource existed, but that did not fix the problem.
