## EDGE STACK RELEASE PROCESS

To be able to run an Ambassador Edge Stack release, you will need clones
of _three_ repos:

- The Ambassador/OSS repo: `https://github.com/datawire/ambassador`

- The Ambassador/Pro repo: `https://github.com/datawire/apro`

- The getambassador.io repo, `https://github.com/datawire/getambassador.io`

Start by cloning both of those. Make sure you can commit and push using your clones.

### FIRST CHECK THE SOURCES OF YAML TRUTH

_NB: this whole section is an annoying crock that will be simplified soon._

The main Source of Truth for the YAML is the Ambassador/OSS repo, in `docs/yaml/ambassador`:

- CRDs are defined in `docs/yaml/ambassador/ambassador-crds.yaml`
- The Ambassador Deployment is the last resource in `docs/yaml/ambassador/ambassador-rbac.yaml`

Beyond that, the rest of an Ambassador Edge Stack deployment is defined in the Ambassador/Pro
repo, in `k8s-aes-src`:

- `k8s-aes-src/00-aes-crds.yaml` has the extra CRDs that Ambassador/OSS does not need, but
  Ambassador Edge Stack does.
- `k8s-aes-src/01-aes.yaml` contains all the resources that Ambassador Edge Stack needs, except 
  for the Ambassador Deployment itself.

**In general, a release of Edge Stack will not require updating YAML.** If it does, the four
files above should have already been updated. If you have any doubt, check with the development
team.

### CREATING THE RELEASE

You can only do RC releases so far. To do one:

0. Go make sure that:

   - Ambassador/Pro `ambassador.commit` points to the correct Ambassador/OSS commit
   - Ambassador/Pro `master` is up-to-date
   - Your clone of getambassador.io is up-to-date
   - Your `EDGE_STACK_UPDATE` environment variable points to your getambassador.io clone:

    ```
    export EDGE_STACK_UPDATE=/path/to/your/getambassador.io/clone
    ```

1. Look up the previous RC tag in the Ambassador/Pro repo. Call `$next` the next RC
   number.

2. Go to your clone of the Ambassador/Pro repo:

    ```
    cd /path/to/apro
    ```

3. Update all YAML as needed:

    ```
    make update-yaml
    ```

4. Check the YAML diffs in Ambassador/Pro:

    ```
    git diff k8s-aes
    ```

5. If there were any YAML diffs, and you're OK with them, commit them on a new branch:

    ```
    git checkout -b rc$next-yaml
    git commit k8s-aes
    git push origin rc$next-yaml
    ```

   and then PR your `rc$next-yaml` branch back into `master`.

6. Check the YAML diffs in getambassador.io:

    ```
    cd $EDGE_STACK_UPDATE
    git diff content
    ```

7. If there were any YAML diffs, and you're OK with them, commit them on a new branch:

    ```
    git checkout -b rc$next-yaml
    git commit content
    git push origin rc$next-yaml
    ```

   and then PR your `rc$next-yaml` branch back into `Edge_stack_update`.

8. Push a tag `v0.99.0-rc$next`:

    ```
    git tag -a v0.99.0-rc$next master
    git push --tags
    ```

9. Check CircleCI status at https://circleci.com/gh/datawire/apro.

10. Wait for CircleCI to show green for your release build before continuing. If the CI build fails,
    figure out why, fix it, and go back to step 0.

### Unneeded? ??

Run 'PROD_KUBECONFIG=<blah> make deploy-aes-backend' to deploy a new aes backend.

### THE MANUAL WAY

Instead of steps 8 - 10 above, you can do

8. **Have a clean tree in your Ambassador/Pro clone.**

9. Do `RELEASE_REGISTRY=quay.io/datawire-dev make rc` in order to release a new image.

10. Make sure your build succeeds.

But don't. Just tag instead.


