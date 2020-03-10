# Edge Control in CI

Imagine you have an application consisting of a hundred microservices and you want to test microservice changes in CI before release. One approach to running such a test would be to spin up a new cluster, install and launch your application in that cluster, replace the one microservice that you wish to test, and run your tests against that cluster. This would work, but it's complicated, expensive, and slow.

If you could use one persistent cluster and application for every test, you would avoid the costs of spinning up a new cluster and installing your application every time. The Telepresence swap deployment workflow lets you do this. Each test run can swap the existing microservice with the modified version running in CI, run tests against the cluster, and then terminate the swap to restore things for the next test. The downside of swap deployment is that the entire cluster is affected, which means that a robust testing strategy must run tests sequentially without overlap. If changes come in fast and test runs are slow, your CI system will fall behind and become the bottleneck.

To avoid this bottleneck, your CI system must be able to test different changesets concurrently without tests interfering with one another. The outbound and intercept features of Edge Control make this possible. The key requirements are

- the microservice being tested must be an HTTP service, and
- it must be possible to set a test-run-specific HTTP header for requests to the microservice being tested. Ideally, that header would pass through from where requests enter the system all the way to the microservice being tested.

The CI job that performs system tests of MyService would look roughly like this

1. Install the required software (`edgectl`, `kubectl`, etc.).
2. Perform the usual CI steps (build the microservice and perform unit/local tests).
3. Set up access to the shared cluster, so that e.g., `kubectl get pods` talks to the right place.
4. Launch the Edge Control Daemon and connect to the cluster
   ```console
   sudo edgectl daemon
   edgectl connect
   ```
5. Add an intercept specific to this test session, e.g., using commit information to construct a unique name
   ```console
   CI_TAG="MyService-$(git describe --always --tags)"
   edgectl intercept add MyService -n "$CI_TAG" -m "x-ci-tag=$CI_TAG" -t localhost:9000
   ```
6. Run system tests by sending requests to the application with the appropriate header set
   ```console
   curl -L -H "x-ci-tag:$CI_TAG" "$API_GATEWAY"
   # (or)
   env "TEST_HEADER_VALUE=$CI_TAG" pytest ...
   # (etc)
   ```
   These requests will go to your shared cluster just as any request would, but calls to MyService from within the application will be routed back to the modified version running in CI because the intercepted header is set appropriately. All other calls to MyService will function as usual, going to the version of MyService running in the cluster.
7. Delete the intercept when the job is done. This is not strictly necessary, as the Traffic Manager will clean up automatically after a while, but doing so here keeps diagnostic output (`edgectl intercept list`, etc.) clean.
   ```console
   edgectl intercept remove "$CI_TAG"
   ```
