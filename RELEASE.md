Run RELEASE_REGISTRY=quay.io/datawire-dev 'make rc' in order to release a new image.

Copy k8s-aes/00-aes-crds.yaml to the Edge-stack-update branch of getambassador.io in <repo>/content/yaml/aes-crds.yaml
Copy k8s-aes/01-aes.yaml to the Edge-stack-update branch of getambassador.io in <repo>/content/yaml/aes.yaml

Run 'PROD_KUBECONFIG=<blah> make deploy-aes-backend' to deploy a new aes backend.
