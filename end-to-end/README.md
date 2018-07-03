Ambassador's end-to-end tests are run by CI, and we will not release an Ambassador for which the end-to-end tests are failing. **You are strongly encouraged to add end-to-end test coverage for features you add.** 

By default, the tests use [`kubernaut`](https://kubernaut.io/) to grab a cluster in which to run the tests. If needed, they will install `kubernaut` for you and walk you through getting a `kubernaut` token. If you want to use your own cluster, set things up so that `kubectl` is talking to your cluster, and set `SKIP_KUBERNAUT` in your environment.

To run the end-to-end tests:

```shell
cd end-to-end
sh testall.sh
```

`testall.sh` will download `kubernaut` if necessary, allocate a `kubernaut` cluster, then run `test.sh` in each `end-to-end` subdirectory with a name that starts with a three-digit number. We will use `$TESTDIR` to refer to a test directory.

The _only_ thing that MUST be present is `$TESTDIR/test.sh`. For each test directory, `testall.sh` will 

```shell
cd $TESTDIR
sh test.sh
```

By convention, `$TESTDIR/k8s` contains Kubernetes manifests that will be applied (using `kubectl apply -f`) to the cluster, and `$TESTDIR/certs` (if present) contains TLS certs for the test. `$TESTDIR/test.sh` is responsible for applying manifests as needed -- most end-to-end tests do _not_ simply apply every manifest in the `k8s` directory at startup.

There are several utilities in the `end-to-end` directory itself; `test.sh` SHOULD typically source these utilities rather than creating one-off versions. Here's a typical `test.sh` start:

```shell
set -e -o pipefail

# Figure out where we are, and get utils.sh imported.
HERE=$(cd $(dirname $0); pwd)

cd "$HERE"

ROOT=$(cd ..; pwd)
PATH="${ROOT}:${PATH}"

source ${ROOT}/utils.sh

# Initialize our cluster. By default, this uses code in 
# kubernaut_utils.sh; see utils.sh for more.
initialize_cluster

# Output cluster info for feedback.
kubectl cluster-info

# Create namespaces, pods, etc. here. This MUST include starting 
# Ambassador itself.
#
# You are encouraged to put everything the test needs into its own 
# namespace(s) -- it makes cleanup MUCH faster.
#
# Once your initial resources have been created...
set +e +o pipefail

# ...wait for all pods to actually be running.
wait_for_pods

# Grab cluster information...
CLUSTER=$(cluster_ip)
APORT=$(service_port ambassador)

# ...and use it to construct a URL that will reach Ambassador.
BASEURL="http://${CLUSTER}:${APORT}"

echo "Base URL $BASEURL"
echo "Diag URL $BASEURL/ambassador/v0/diag/"

# Wait for Ambassador to actually be ready.
wait_for_ready "$BASEURL"
```

This is fairly typical setup code. Other utility functions are available for checking Ambassador weights, etc., but we're also at the point where we should probably just switch to Python if much more has to be added.

Note well:
- You will need to use Python 3 to run the tests.
- The tests will not work unless you've pushed your build to a public `DOCKER_REGISTRY`. See [building instructions](../BUILDING.md).

If you're working with this stuff, we'd love to see you in our [Slack channel](https://d6e.co/slack)!
