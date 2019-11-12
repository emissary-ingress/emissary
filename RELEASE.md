To do an RC release right now:

0. Go make sure that `ambassador.commit` has a good commit ID, and that 
   `master` is up-to-date with what you want to release.

1. Do either **THE CI WAY** or **THE MANUAL WAY** below.

### THE CI WAY

2. Push tag `v0.10.0-rc$next` where `$next` is one more than the previous RC tag.

3. Check CircleCI status at https://circleci.com/gh/datawire/apro.

4. When CircleCI shows green for your release build, continue with **UPDATE YAML** below.

### THE MANUAL WAY

2. **Have a clean tree.**

3. Do `RELEASE_REGISTRY=quay.io/datawire-dev make rc` in order to release a new image.

4. Make sure your build succeeds.

### UPDATE YAML

5. Go your clone of datawire/getambassador.io.git, and make a new branch off `Edge-stack-update`.

6. Let `$EDGE_STACK_UPDATE` be the path to your clone of datawire/getambassador.io.git. From your apro clone:

    ```
    cp k8s-aes/00-aes-crds.yaml $EDGE_STACK_UPDATE/content/yaml/aes-crds.yaml

    sed -e 's!{{env "AES_IMAGE"}}!quay.io/datawire-dev/aes:0.99.0-rc-latest!' \
      < k8s-aes/01-aes.yaml \
      > $EDGE_STACK_UPDATE/content/yaml/aes.yaml
    ```

7. PR your new branch back into `Edge-stack-update`.

### Unneeded? ??

Run 'PROD_KUBECONFIG=<blah> make deploy-aes-backend' to deploy a new aes backend.
