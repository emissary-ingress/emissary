To do an RC release right now:

1. *Have a clean tree.*

2. Do `RELEASE_REGISTRY=quay.io/datawire-dev make rc` in order to release a new image.

3. Go your clone of datawire/getambassador.io.git, and make a new branch off `Edge-stack-update`.

4. Let `$EDGE_STACK_UPDATE` be the path to your clone of datawire/getambassador.io.git. From your apro clone:

   - `cp k8s-aes/00-aes-crds.yaml $EDGE_STACK_UPDATE/content/yaml/aes-crds.yaml`
   - `sed -e 's!{{env "AES_IMAGE"}}!quay.io/datawire-dev/aes:0.10.0-rc-latest!' < k8s-aes/01-aes.yaml > $EDGE_STACK_UPDATE/content/yaml/aes.yaml`

Run 'PROD_KUBECONFIG=<blah> make deploy-aes-backend' to deploy a new aes backend.
