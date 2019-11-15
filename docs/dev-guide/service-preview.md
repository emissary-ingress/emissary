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

### Create a Service

Create the following YAML and put it in a file called `tour.yaml`.

  <div class="gatsby-highlight" data-language="yaml">
  <pre class="language-yaml">
  <code class="language-yaml" id="step7">
  ---
  apiVersion: v1
  kind: Service
  metadata:
    name: tour
  spec:
    ports:
    - name: ui
      port: 5000
      targetPort: 5000
    - name: backend
      port: 8080
      targetPort: 8080
    selector:
      app: tour
  ---
  apiVersion: apps/v1
  kind: Deployment
  metadata:
    name: tour
  spec:
    replicas: 1
    selector:
      matchLabels:
        app: tour
    strategy:
      type: RollingUpdate
    template:
      metadata:
        labels:
          app: tour
      spec:
        containers:
        - name: tour-ui
          image: quay.io/datawire/tour:ui-$tourVersion$
          ports:
          - name: http
            containerPort: 5000
        - name: quote
          image: quay.io/datawire/tour:backend-$tourVersion$
          ports:
          - name: http
            containerPort: 8080
          resources:
            limits:
              cpu: "0.1"
              memory: 100Mi
  ---
  apiVersion: getambassador.io/v2
  kind: Mapping
  metadata:
    name: tour-ui
  spec:
    prefix: /
    service: tour:5000
  ---
  apiVersion: getambassador.io/v2
  kind: Mapping
  metadata:
    name: tour-backend
  spec:
    prefix: /backend/
    service: tour:8080
    labels:
      ambassador:
        - request_label:
          - backend
  </code></pre></div>
  <p>
  <p>
  <button onclick="copy_to_clipboard('step7')">Copy to Clipboard</button>

Then, apply it to the Kubernetes with `kubectl`:

  <div class="gatsby-highlight" data-language="shell">
  <pre class="language-shell">
  <code class="language-shell" id="step8">
  kubectl apply -f tour.yaml</code></pre></div>

<button onclick="copy_to_clipboard('step8')">Copy to Clipboard</button>

This YAML has also been published so you can deploy it remotely:

  <div class="gatsby-highlight" data-language="shell">
  <pre class="language-shell">
  <code class="language-shell" id="step9">
  kubectl apply -f https://getambassador.io/yaml/tour/tour.yaml</code></pre></div>


<button onclick="copy_to_clipboard('step9')">Copy to Clipboard</button>

When the `Mapping` CRDs are applied, Ambassador will use them to configure routing:

- The first `Mapping` causes traffic from the `/` endpoint to be routed to the `tour-ui` React application.
- The second `Mapping` causes traffic from the `/backend/` endpoint to be routed to the `tour-backend` service.

Note also the port numbers in the `service` field of the `Mapping`. This allows us to use a single service to route to both the containers running on the `tour` pod.

<span style="color:#f9634E">**Important:**</span>

Routing in Ambassador Open Source can be configured with Ambassador OSS resources as shown above, Kubernetes service annotation, and Kubernetes Ingress resources.

Ambassador OSS ustom resources are the recommended config format and will be used throughout the documentation.

See [configuration format](/reference/config-format) for more information on your configuration options.

### Test the Mapping

To test things out, we'll need the external IP for Ambassador (it might take some time for this to be available):

<div class="gatsby-highlight" data-language="shell">
<pre class="language-shell">
<code class="language-shell" id="step10">
kubectl get svc -o wide ambassador</code></pre></div>
<p>
<p>
<button onclick="copy_to_clipboard('step10')">Copy to Clipboard</button>

Eventually, this should give you something like:

```
NAME         CLUSTER-IP      EXTERNAL-IP     PORT(S)        AGE
ambassador   10.11.12.13     35.36.37.38     80:31656/TCP   1m
```


You should now be able to reach the `tour-ui` application from a web browser:

http://35.36.37.38/

or on minikube:

```shell
$ minikube service list
|-------------|----------------------|-----------------------------|
|  NAMESPACE  |         NAME         |             URL             |
|-------------|----------------------|-----------------------------|
| default     | ambassador-admin     | http://192.168.99.107:30319 |
| default     | ambassador           | http://192.168.99.107:31893 |
|-------------|----------------------|-----------------------------|
```
http://192.168.99.107:31893/

or on Docker for Mac/Windows:

```shell
$ kubectl get svc
NAME               TYPE           CLUSTER-IP       EXTERNAL-IP   PORT(S)          AGE
ambassador         LoadBalancer   10.106.108.64    localhost     80:32324/TCP     13m
ambassador-admin   NodePort       10.107.188.149   <none>        8877:30993/TCP   14m
tour               ClusterIP      10.107.77.153    <none>        80/TCP           13m
kubernetes         ClusterIP      10.96.0.1        <none>        443/TCP          84d
```
http://localhost/

### Run Locally and Clone 

We're first going to get the tour backend service running locally. Clone the tour repository and build a local image.

    ```
    git clone https://github.com/datawire/tour
    cd tour
    make docker.run
    ```

    Note that Preview doesn't depend on a locally running container; you can just run a service locally on your laptop. We're using a container in this tutorial to minimize environmental issues with different golang environments.

    In the `make` command above, we build the backend application in a docker container named `localhost:31000/tour:backend-latest` and run it on port 8080.

### Redeploy Tour with Preview Sidecar

Now, in another terminal window, redeploy the tour application with the Preview sidecar. The sidecar is special process which will route requests to your local machine or to the production cluster. The `apictl traffic inject` command will automatically create the appropriate YAML to inject the sidecar. In the `tour` directory, pass the file name of the QOTM deployment:

   ```
   apictl traffic inject k8s/tour.yaml -d tour -s tour -p 8080 > k8s/tour-traffic-sidecar.yaml
   ```

   This will create a YAML file called `qotm-sidecar.yaml`. The file will look like the following:

   ```yaml
    ---
    apiVersion: getambassador.io/v2
    kind: Mapping
    metadata:  
      name: tour-ui
    spec:
      prefix: /
      service: tour:5000
    ---
    apiVersion: getambassador.io/v2
    kind: Mapping
    metadata:  
      name: tour-backend
    spec:
      prefix: /backend/
      service: tour:8080
        labels:
          ambassador:
            - request_label:
              - backend
    ---
    apiVersion: v1
    kind: Service
    metadata:
      name: tour
    spec:
      ports:
      - name: ui
        port: 5000
        targetPort: 5000
      - name: backend
        port: 8080
        targetPort: 9900
      selector:
        app: tour
    ---
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: tour
    spec:
      replicas: 1
      selector:
        matchLabels:
          app: tour
      strategy:
        type: RollingUpdate
      template:
        metadata:
          labels:
            app: tour
        spec:
          containers:
          - image: quay.io/datawire/tour:ui-$tourVersion$
            name: tour-ui
            ports:
            - containerPort: 5000
              name: http
          - image: quay.io/datawire/tour:backend-$tourVersion$
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
              value: tour
            - name: APPPORT
              value: "8080"
            - name: AMBASSADOR_LICENSE_KEY
              value: eJcbGciOiaIUzI1NiIsInR5cCkpXVCJ9.eCI6Im5rcmF1c2UiLCJleHAiOjE1Nzg0MTg4ODZ9.S_6-zdPyy4z1N4Jmo5e4A7fME4CbQVL_13ikw
            image: quay.io/datawire/ambassador_pro:app-sidecar-$aproVersion$
            name: traffic-sidecar
            ports:
            - containerPort: 9900
   ```

   If you examine this file, you will notice a couple of important difference:
   - The `traffic-sidecar` container has been added to the deployment
   - In the service, `targetPort` for the `backend` port mapping has been changed to point to port 9900 which is the port the `traffic-sidecar` container is listening on

### Redeploy with Sidecar

Then, redeploy tour with the sidecar:

   ```
   kubectl apply -f k8s/tour-traffic-sidecar.yaml
   ```

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

### Create an Intercept Rule

We need to create an `intercept` rule that tells Ambassador Edge Stack where to route specific requests. The following command will tell Ambassador Edge Stack to route any traffic for the `tour` deployment where the header `x-service-preview` is `dev` to go to port 8080 on localhost:



    ```
    apictl traffic intercept tour -n x-service-preview -m dev -t 8080
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
