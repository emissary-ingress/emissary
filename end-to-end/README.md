Run `testall.sh` in this directory to descend into each 0* directory and run the tests.

By default, the tests use [`kubernaut`](https://kubernaut.io/) to grab a cluster in which to run the tests. If needed, they will install `kubernaut` for you and walk you through getting a `kubernaut` token. If you want to use your own cluster, set things up so that `kubectl` is talking to your cluster, and set SKIP_KUBERNAUT in your environment.

Note well:
- You will need to use Python 3 to run the tests.
- The tests will not work unless you've pushed your build to a public `DOCKER_REGISTRY`. See [building instructions](../BUILDING.md).

If you're working with this stuff, we'd love to see you in our [Gitter channel](https://gitter.im/datawire/ambassador)!
