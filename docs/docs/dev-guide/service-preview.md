# Service Preview

**This page is deprecated. For the latest version, check out [Edge Control](../../../reference/edge-control).**

How do you verify that the code you've written actually works? Ambassador's *Service Preview* lets developers see exactly how their service works in a realistic environment -- without impacting other developers or end-users. Service Preview integrates [Telepresence](https://www.telepresence.io), the popular CNCF project for local development and debugging on Kubernetes.

## Install `apictl`

`apictl` is the command client for Ambassador.

Download the latest version of the client:

<a class="apictl-dl" href="https://s3.amazonaws.com/datawire-static-files/apictl/$aproVersion$/darwin/amd64/apictl">Mac 64-bit</a> |
<a class="apictl-linux-dl" href="https://s3.amazonaws.com/datawire-static-files/apictl/$aproVersion$/linux/amd64/apictl">Linux 64-bit</a>

Make sure the client is somewhere on your PATH. In addition, place your license key in `~/.ambassador.key`.

Information about open-source code used in `apictl` can be found by running `apictl --version`.

## Getting Started

In this quick start, we're going to preview a change we make to the backend service of the quote application, without impacting normal users of the application. Before getting started, make sure the [quote application is installed](../../../user-guide/getting-started#create-a-mapping) on your cluster and you've installed the `apictl` command-line tool, as explained above.

1. We're first going to get the quote backend service running locally. Clone the quote repository and build a local image.

    ```
    git clone https://github.com/datawire/quote
    cd quote
    make docker.run
    ```

    Note that Preview doesn't depend on a locally running container; you can just run a service locally on your laptop. We're using a container in this tutorial to minimize environmental issues with different Golang environments.

    In the `make` command above, we build the backend application in a docker container named `localhost:31000/quote:backend-latest` and run it on port 8080.

2. Now, in another terminal window, redeploy the quote application with the Preview sidecar. The sidecar is a special process that will route requests to your local machine or the production cluster. The `apictl traffic inject` command will automatically create the appropriate YAML to inject the sidecar. In the `quote` directory, pass the file name of the QOTM deployment:

   ```
   apictl traffic inject k8s/tour.yaml -d tour -s tour -p 8080 > k8s/tour-traffic-sidecar.yaml
   ```

   This will create a YAML file called `qotm-sidecar.yaml`. The file will look like the following:

   ```yaml
    ---
    apiVersion: getambassador.io/v2
    kind: Mapping
    metadata:  
      name: quote-backend
    spec:
      prefix: /backend/
      service: quote:8080
        labels:
          ambassador:
            - request_label:
              - backend
    ---
    apiVersion: v1
    kind: Service
    metadata:
      name: quote
    spec:
      ports:
      - name: backend
        port: 8080
        targetPort: 9900
      selector:
        app: quote
    ---
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: quote
    spec:
      replicas: 1
      selector:
        matchLabels:
          app: quote
      strategy:
        type: RollingUpdate
      template:
        metadata:
          labels:
            app: quote
        spec:
          containers:
          - image: quay.io/datawire/quote:$quoteVersion$
            name: quote
            ports:
            - containerPort: 8080
              name: http
            resources:
              limits:
                cpu: "0.1"
                memory: 100Mi
          - env:
            - name: APPNAME
              value: quote
            - name: APPPORT
              value: "8080"
            - name: AMBASSADOR_LICENSE_KEY
              value: eJcbGciOiaIUzI1NiIsInR5cCkpXVCJ9.eCI6Im5rcmF1c2UiLCJleHAiOjE1Nzg0MTg4ODZ9.S_6-zdPyy4z1N4Jmo5e4A7fME4CbQVL_13ikw
            image: quay.io/datawire/aes:$version$
            command: ["app-sidecar"]
            name: traffic-sidecar
            ports:
            - containerPort: 9900
   ```

   If you examine this file, you will notice a couple of important difference:
   - The `traffic-sidecar` container has been added to the deployment
   - In the service, `targetPort` for the `backend` port mapping has been changed to point to port 9900 which is the port the `traffic-sidecar` container is listening on

4. Redeploy quote with the sidecar:

   ```shell
   kubectl apply -f k8s/quote-traffic-sidecar.yaml
   ```

5. Test to make sure that both your production and development instances of QOTM work:

    ```
    curl $AMBASSADOR_IP/backend/ # test production
    curl localhost:8080/         # test development
    ```

6. Initialize the traffic manager for the cluster.

    ```
    apictl traffic initialize
    ```

7. We need to create an `intercept` rule that tells Ambassador where to route specific requests. The following command will tell Ambassador to route any traffic for the `quote` deployment where the header `x-service-preview` is `dev` to go to port 8080 on localhost:

    ```
    apictl traffic intercept quote -n x-service-preview -m dev -t 8080
    ```

8. Requests with the header `x-service-preview: dev` will now get routed locally:

    ```
    curl -H "x-service-preview: dev" $AMBASSADOR_IP/backend/` # will go to local Docker instance
    curl $AMBASSADOR_IP/backend/                              # will go to production instance
    ```

9. Make a change to the backend source code. In `backend/main.go`, uncomment out line 85, and comment out line 84, so it reads like so:

    ```golang
    ...
    //quote := s.random.RandomSelectionFromStringSlice(s.quotes)
    quote := "Service Preview Rocks!"
    ...
    ```

    This will ensure that the backend service will return a quote of "Service Preview rocks" every time.

10. Rebuild the docker container and rerun  the `curl` above, which will now route to your (modified) local copy of the QOTM service:

    ```
    make docker.run -C backend/
    curl -H "x-service-preview: dev" $AMBASSADOR_IP/qotm/` # will go to local Docker instance
    ```

    To recap: With Preview, we can now see test and visualize changes to our service that we've made locally, without impacting other users of the stable version of that service.

## Using Service Preview

Service Preview will match HTTP headers based on the headers that are seen by the *sidecar* and not the edge gateway. Matches are made on the whole header, e.g., a match rule of `dev` will not match in the example above, while `/backend/dev` will match.

While any HTTP header will match, in practice, using host-based routing (i.e., the `:authority` header), a custom HTTP header (e.g., the `x-service-preview` header used above), or an authentication header is recommended.

This line has been added for testing purposes.
