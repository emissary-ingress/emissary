# Service Preview

How do you verify that the code you've written actually works? Ambassador Pro's *Service Preview* lets developers see exactly how their service works in a realistic enviroment -- without impacting other developers or end users. Service Preview integrates [Telepresence](https://www.telepresence.io), the popular CNCF project for local development and debugging on Kubernetes.

## Install `apictl`

`apictl` is the command client for Ambassador Pro.

Download the latest version of the client:

<a class="apictl-dl" href="https://s3.amazonaws.com/datawire-static-files/apictl/%aproVersion%/darwin/amd64/apictl">Mac 64-bit</a> |
<a class="apictl-linux-dl" href="https://s3.amazonaws.com/datawire-static-files/apictl/%aproVersion%/linux/amd64/apictl">Linux 64-bit</a>

Make sure the client is somewhere on your PATH. In addition, place your license key in `~/.ambassador.key`.

## Getting started

In this quick start, we're going to preview a change we make to the QOTM service, without impacting normal users of the QOTM service. Before getting started, make sure the [QOTM service is installed](https://www.getambassador.io/user-guide/getting-started#5-adding-a-service) on your cluster and you've installed the `apictl` command line tool, as explained above.

1. We're first going to get the QOTM service running locally. Clone the QOTM repository and build a local Docker image.

    ```
    git clone https://github.com/datawire/qotm
    cd qotm
    docker build . -t qotm:dev
    docker run --rm -it -v $(pwd):/service -p 5000:5000 qotm:dev
    ```

    Note that Preview doesn't depend on a locally running container; you can just run a service locally on your laptop. We're using a container in this tutorial to minimize environmental issues with different Python environments.

    In the `docker run` command above, we mount the local directory into the container, so that any code changes to the QOTM service happen immediately. 

2. Now, in another terminal window, redeploy the QOTM service with the Preview sidecar. The sidecar is special process which will route requests to your local machine or to the production cluster. The `apictl traffic inject` command will automatically create the appropriate YAML to inject the sidecar. In the `qotm` directory, pass the file name of the QOTM deployment:

   ```
   apictl traffic inject kubernetes/qotm-deployment.yaml -d qotm -p 5000 > qotm-sidecar.yaml
   ```

   This will create a YAML file called `qotm-sidecar.yaml`. The file will look like the following:

   ```
   apiVersion: extensions/v1beta1
   kind: Deployment
   metadata:
     name: qotm
   spec:
     replicas: 1
     strategy:
       type: RollingUpdate
     template:
       metadata:
        labels:
           app: qotm
       spec:
         containers:
         - name: qotm
           image: datawire/qotm:1.2
           ports:
           - name: http-api
             containerPort: 5000
           resources:
             limits:
               cpu: "0.1"
               memory: 100Mi
         - env:
           - name: APPNAME
             value: qotm
           - name: APPPORT
             value: "5000"
           image: quay.io/datawire/ambassador_pro:app-sidecar-%aproVersion%
           name: traffic-sidecar
           ports:
           - containerPort: 9900
   ```

3. Update the QOTM service YAML to point to the sidecar on port 9900, instead of the QOTM service directly on port 5000.

   ```yaml
   ---
   apiVersion: v1
   kind: Service
   metadata:
     name: qotm
     annotations:
       getambassador.io/config: |
         ---
         apiVersion: ambassador/v1
         kind:  Mapping
         name:  qotm_mapping
         prefix: /qotm/
         service: qotm
   spec:
      selector:
        app: qotm
   ports:
      - port: 80
      name: http-qotm
      targetPort: 9900
   ```

4. Redeploy QOTM with the sidecar:

   ```
   kubectl apply -f qotm-sidecar.yaml
   kubectl apply -f qotm-service.yaml
   ```

5. Test to make sure that both your production and development instances of QOTM work:

    ```
    curl $AMBASSADOR_IP/qotm/ # test production
    curl localhost:5000       # test development
    ```

6. Initialize the traffic manager for the cluster.

    ```
    apictl traffic initialize
    ```

7. We need to create an `intercept` rule that tells Ambassador where to route specific requests. The following command will tell Ambassador to route any traffic for the `qotm` deployment where the header `x-service-preview` is `dev` to go to port 5000 on localhost:

    ```
    apictl traffic intercept qotm -n x-service-preview -m dev -t 5000
    ```

8. Requests with the header `x-service-preview: dev` will now get routed locally:

    ```
    curl -H "x-service-preview: dev" $AMBASSADOR_IP/qotm/` # will go to local Docker instance
    curl $AMBASSADOR_IP/qotm/ # will go to production instance
    ```

9. Make a change to the QOTM source code. In `qotm/qotm.py`, uncomment out line 149, and comment out line 148, so it reads like so:

    ```
    return RichStatus.OK(quote="Telepresence rocks!")
    # return RichStatus.OK(quote=random.choice(quotes))
    ```

    This will insure that the QOTM service will return a quote of "Telepresence rocks" every time.

10. Re-run the `curl` above, which will now route to your (modified) local copy of the QOTM service:

    ```
    curl -H "x-service-preview: dev" $AMBASSADOR_IP/qotm/` # will go to local Docker instance
    ```

    To recap: With Preview, we can now see test and visualize changes to our service that we've mode locally, without impacting other users of the stable version of that service.

## Using Service Preview

Service Preview will match HTTP headers based on the headers that are seen by the *sidecar*, and not the edge gateway. Matches are made on the whole header, e.g., a match rule of `dev` will not match in the example above, while `/qotm/dev` will match.

While any HTTP header will match, in practice, using host-based routing (i.e., the `:authority` header), a custom HTTP header (e.g., the `x-service-preview` header used above), or an authentication header is recommended.