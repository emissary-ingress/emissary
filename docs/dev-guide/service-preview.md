# Service Preview

How do you verify that the code you've written actually works? Ambassador Pro's *Service Preview* lets developers see exactly how their service works in a realistic enviroment -- without impacting other developers or end users. Service Preview integrates [Telepresence](https://www.telepresence.io), the popular CNCF project for local development and debugging on Kubernetes.

## Getting started

In this quick start, we're going to preview a change we make to the QOTM service, without impacting normal users of the QOTM service. Before getting started, make sure:

* The [QOTM service is installed](https://www.getambassador.io/user-guide/getting-started#5-adding-a-service) on your cluster.
* You've installed the `apictl` command line tool, as explained in the [Ambassador Pro installation documentation](https://www.getambassador.io/user-guide/ambassador-pro-install).

1. We're first going to get the QOTM service running locally. Clone the QOTM repository and build a local Docker image.

    ```
    git clone https://github.com/datawire/qotm
    cd qotm
    docker build . -t qotm:dev
    docker run --rm -it -v $(pwd):/service -p 5000:5000 qotm:dev
    ```

    Note that Preview doesn't depend on a locally running container; you can just run a service locally on your laptop. We're using a container in this tutorial to minimize environmental issues with different Python environments.

    In the `docker run` command above, we mount the local directory into the container, so that any code changes to the QOTM service happen immediately. 

2. Now, in another terminal window, redeploy the QOTM service with the Preview sidecar. The sidecar is special process which will route requests to your local machine or to the production cluster. The `service` and `deployment` is updated as per below.

    ```
---
apiVersion: v1
kind: Service
metadata:
  name: qotm
  annotations:
    getambassador.io/config: |
      ---
      apiVersion: ambassador/v0
      kind:  Mapping
      name:  qotm_mapping
      prefix: /qotm/
      service: qotm
spec:
  selector:
    app: qotm
  ports:
    - port: 80
      targetPort: 9900
  type: ClusterIP
---
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
        image: datawire/qotm:1.1
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
        image: ark3/telepresence-sidecar:18
        name: traffic-sidecar
        ports:
        - containerPort: 9900
    ```

3. Test to make sure that both your production and development instances of QOTM work:

    ```
    curl $AMBASSADOR_IP/qotm/ # test production
    curl localhost:5000       # test development
    ```

4. Initialize the traffic manager for the cluster.

    ```
    apictl traffic initialize
    ```

5. We need to create an `intercept` rule that tells Ambassador where to route specific requests. Enter the following:

    ```
    apictl traffic intercept qotm -n :path -m /dev -t 5000
    ```

6. Requests to `/dev` will now get routed locally:

    ```
    curl $AMBASSADOR_IP/dev` # will go to local Docker instance
    curl $AMBASSADOR_IP/qotm/ # will go to production instance
    ```

7. Make a change to the QOTM source code. In `qotm/qotm.py`, uncomment out line 149, and comment out line 148, so it reads like so:

    ```
    return RichStatus.OK(quote="Telepresence rocks!")
    # return RichStatus.OK(quote=random.choice(quotes))
    ```

    This will insure that the QOTM service will return a quote of "Telepresence rocks" every time.

8. Re-run the `curl` above, which will now route to your (modified) local copy of the QOTM service:

   ```
   curl $AMBASSADOR_IP/dev` # will go to local Docker instance
   ```

   To recap: With Preview, we can now see test and visualize changes to our service that we've mode locally, without impacting other users of the stable version of that service.

