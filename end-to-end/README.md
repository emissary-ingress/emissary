Run `testall.sh` in this directory to descend into each 0* directory and run the tests.

#### Requires:

* Python 3
* [kubernaut](https://github.com/datawire/kubernaut#installation)
* kubernaut token from https://kubernaut.io/token
* A build pushed to a public DOCKER_REGISTRY. See [building instructions](../BUILDING.md).

Note that running the end-to-end tests will install [kubernaut](kubernaut.io) as `end-to-end/kubernaut`.
