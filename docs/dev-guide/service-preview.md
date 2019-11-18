# Service Preview

How do you verify that the code you've written actually works? Ambassador Edge Stack's *Service Preview* lets developers see exactly how their service works in a realistic enviroment -- without impacting other developers or end users. Service Preview integrates [Telepresence](https://www.telepresence.io), the popular CNCF project for local development and debugging on Kubernetes.

## Install `apictl`

`apictl` is the command client for Ambassador Edge Stack.


Download the latest version of the client:

<a class="apictl-dl" href="https://s3.amazonaws.com/datawire-static-files/apictl/$aproVersion$/darwin/amd64/apictl">Mac 64-bit</a> |
<a class="apictl-linux-dl" href="https://s3.amazonaws.com/datawire-static-files/apictl/$aproVersion$/linux/amd64/apictl">Linux 64-bit</a>

Make sure the client is somewhere on your PATH. In addition, place your license key in `~/.ambassador.key`.

Information about open source code used in `apictl` can be found by running `apictl --version`.

## Deploy a Service

In this quick start, we're going to create an application and then add a backend service to the application, without impacting normal users of the application.

### Test QOTM

Test to make sure that both your production and development instances of QOTM work:

    ```
    curl $AMBASSADOR_IP/backend/ # test production
    curl localhost:8080/         # test development
    ```

### Initialize Traffic Manager

 Initialize the traffic manager for the cluster.

    ```
    apictl traffic initialize
    ```


### Route Locally

Requests with the header `x-service-preview: dev` will now get routed locally:

    ```
    curl -H "x-service-preview: dev" $AMBASSADOR_IP/backend/` # will go to local Docker instance
    curl $AMBASSADOR_IP/backend/                              # will go to production instance
    ```

### Change Backend Code

Make a change to the backend source code. In `backend/main.go`, uncomment out line 85, and comment out line 84, so it reads like so:

    ```golang
    ...
    //quote := s.random.RandomSelectionFromStringSlice(s.quotes)
    quote := "Service Preview Rocks!"
    ...
    ```

    This will insure that the backend service will return a quote of "Service Preview rocks" every time.

### Rebuild the Container

Rebuild the docker container and rerun  the `curl` above, which will now route to your (modified) local copy of the QOTM service:

    ```
    make docker.run -C backend/
    curl -H "x-service-preview: dev" $AMBASSADOR_IP/qotm/` # will go to local Docker instance
    ```

    To recap: With Preview, we can now see test and visualize changes to our service that we've mode locally, without impacting other users of the stable version of that service.

## Using Service Preview

Service Preview will match HTTP headers based on the headers that are seen by the *sidecar*, and not the edge gateway. Matches are made on the whole header, e.g., a match rule of `dev` will not match in the example above, while `/backend/dev` will match.

While any HTTP header will match, in practice, using host-based routing (i.e., the `:authority` header), a custom HTTP header (e.g., the `x-service-preview` header used above), or an authentication header is recommended.
