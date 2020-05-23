# OLD TESTS HERE. NO LONGER USED. DO NOT READ.

**These tests are no longer used. Do not reference them.**

Ambassador's end-to-end tests are run by CI, and we will not release an Ambassador for which the end-to-end tests are failing. **You are strongly encouraged to add end-to-end test coverage for features you add.** 

By default, the tests use [`kubernaut`](https://kubernaut.io/) to grab a cluster in which to run the tests. If needed, they will install `kubernaut` for you and walk you through getting a `kubernaut` token. If you want to use your own cluster, set things up so that `kubectl` is talking to your cluster, and set `SKIP_KUBERNAUT` in your environment.

#### Running end-to-end tests

The `e2e` target in the Makefile will run all your end-to-end tests.
It will download `kubernaut` if necessary, allocate a `kubernaut` cluster, then run `test.sh` in each `end-to-end` subdirectory with a name that starts with a three-digit number. We will use `$TESTDIR` to refer to a test directory. The _only_ thing that MUST be present is `$TESTDIR/test.sh`.

- Running all tests

  ```
  make e2e
  ```
  
- Running a single test, say `001-simple`

  ```
  make E2E_TEST_NAME=001-simple e2e
  ```

- Running tests with a specific Ambassador image

  This is particularly useful when you have only made changes to the end-to-end tests

  ```
  make AMBASSADOR_DOCKER_IMAGE=docker.io/datawire/ambassador:0.37.0 e2e
  ```

- Clean up test artifacts (among other generated junk)
  ```
  make clobber
  ```

#### Writing end-to-end tests

By convention, `$TESTDIR/k8s` contains Kubernetes manifests that will be applied (using `kubectl apply -f`) to the cluster, and `$TESTDIR/certs` (if present) contains TLS certs for the test. `$TESTDIR/test.sh` is responsible for applying manifests as needed -- most end-to-end tests do _not_ simply apply every manifest in the `k8s` directory at startup.

There are several utilities in the `end-to-end` directory itself; `test.sh` SHOULD typically source these utilities rather than creating one-off versions. Here's a typical `test.sh`:

```shell

set -e -o pipefail

# Set the namespace this test will run in. This is generally the name of the test itself.
NAMESPACE="016-empty-yaml"

# Figure out where we are, and source utils.sh to use the functions in there
cd $(dirname $0)
ROOT=$(cd ../..; pwd)
source ${ROOT}/utils.sh

# Bootstraps the namespace and configures kubectl command for further use
bootstrap --cleanup ${NAMESPACE} ${ROOT}

# Patch the default Ambassador deployment file with AMBASSADOR_ID and namespace as set above
python ${ROOT}/yfix.py ${ROOT}/fixes/test-dep.yfix \
    ${ROOT}/ambassador-deployment.yaml \
    k8s/ambassador-deployment.yaml \
    ${NAMESPACE} \
    ${NAMESPACE}

# Create namespaces, pods, etc. here. This MUST include starting 
# Ambassador itself.
#
# You are encouraged to put everything the test needs into its own 
# namespace(s) -- it makes cleanup MUCH faster.
#
# Once your initial resources have been created...
kubectl apply -f k8s/rbac.yaml
kubectl apply -f k8s/ambassador.yaml
kubectl apply -f k8s/qotm.yaml
kubectl apply -f k8s/ambassador-deployment.yaml

set +e +o pipefail

# Wait for all the pods to come up in this namespace
wait_for_pods ${NAMESPACE}

# IP of the cluster Ambassador is running in
CLUSTER=$(cluster_ip)
# The first service port that Ambassador is listening on
APORT=$(service_port ambassador ${NAMESPACE})

# Generate the URL where Ambassador can be reached
BASEURL="http://${CLUSTER}:${APORT}"

echo "Base URL $BASEURL"
echo "Diag URL $BASEURL/ambassador/v0/diag/"

# Wait till Ambassador is ready and ready to serve
wait_for_ready "$BASEURL" ${NAMESPACE}

# Check the diag file matched the diag files from ambassador
if ! check_diag "$BASEURL" 1 "No annotated services"; then
    exit 1
fi

# Delete the namespace upon completion of the test
if [ -n "$CLEAN_ON_SUCCESS" ]; then
    drop_namespace ${NAMESPACE}
fi
```

This is a fairly typical setup code. Other utility functions are available for checking Ambassador weights, etc., but we're also at the point where we should probably just switch to Python if much more has to be added.

Note well:
- You will need to use Python 3 to run the tests.
- The tests will not work unless you've pushed your build to a public `DOCKER_REGISTRY`. See [building instructions](../BUILDING.md).

Tip:
When writing a new test, it's best to iterate over an existing test. Tests like [000-no-base](1-parallel/000-no-base/test.sh) and [016-empty-yaml](1-parallel/016-empty-yaml/test.sh) are great candidates to start iterating from.

###### Important points
- When you run `bootstrap` function at the starting of a test, the `kubectl` command is aliased to `kubectl -n <namespace>`. This means that you do not have to worry about specifying namespace in your Kubernetes artifacts under `k8s/` directory. However, if you wish to deploy to a separate namespace, then you will need to prefix your kubectl command with `\`. The command will look like `\kubectl apply -f ...`. This will make bash ignore any `kubectl` alias and execute the command as is. 

- Make sure you add an `k8s/rbac.yaml` to the test you're writing, and apply it in test.sh `kubectl apply -f rbac.yaml`. The file will look something like -
```yaml
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: ambassador
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: ambassador-<namespace>
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: ambassador
subjects:
- kind: ServiceAccount
  name: ambassador
  namespace: <namespace>
```

- Make sure all of your Ambassador annotations have `ambassador_id` set. This is important because multiple tests are executed in parallel and for configurations to apply to a specific Ambassador instance, `ambassador_id` is used.

If you're working with this stuff, we'd love to see you in our [Slack channel](https://d6e.co/slack)!
