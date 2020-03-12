# Edge Control

Edge Control is the architecture for `edgectl` and allows you to provide the functionality for application users and ease of use for your cluster modification. With Edge Control, developers can safely share a single development cluster, and enable the "always-on" experience originally achieved with custom workarounds. For your users, Edge Control allows you to provide basic and advanced functionality to your application with minimal cluster-side modifications.

You can use Edge Control for developing new services, and debugging existing services.

**New Service**: If you are a developer and you want to write a new service, it depends on existing services running in your cluster. You can use the `edgectl connect` command to set up outbound connectivity from your laptop to your cluster. This allows the work-in-progress implementation of your new service to connect to existing services on your laptop.

**Debugging Existing Services**: If you need to test a bug fix for an existing service running in the cluster, you can use `edgectl intercept`. Designate a subset of requests for this service as intercepted, which will then be redirected to your laptop. You can then run a modified implementation of the service to test the bug fix. All other requests will go to the existing service running in your cluster without disruption.

To start using Edge Control:

* Install on a Laptop
* Install in a Cluster
* Configure Outbound Services
* Intercept Requests for Bugging

## Install Edge Control: Laptop

1. Grab the latest `edgectl` executable and install it somewhere in your shell’s PATH.

For MacOS:

```bash
curl -fLO https://metriton.datawire.io/downloads/darwin/edgectl
chmod a+x edgectl
mv edgectl ~/bin  # Somewhere in your PATH
```

For Linux:

```bash
curl -fLO https://metriton.datawire.io/downloads/linux/edgectl
chmod a+x edgectl
mv edgectl ~/bin  # Somewhere in your PATH
```

Note: Similar instructions work for Windows:

```bash
curl -fLO https://metriton.datawire.io/downloads/windows/edgectl.exe
mv edgectl.exe C:\windows\  # Somewhere in your PATH
```

but Edge Control’s cluster features, as described in this document, do not work correctly on Windows at this time.

Note: You can build Edge Control from source, but the straightforward way

```bash
GO111MODULE=on go get github.com/datawire/ambassador/cmd/edgectl`
```

leaves you with a binary that has no embedded version number. If you really want to build from source, clone the repository and run `./builder/build_push_cli.sh build`, which will leave a binary in the `~/bin directory`. We will have a better answer for building from source soon.

2. Launch the daemon component using `sudo`:

```bash
$ sudo edgectl daemon
Launching Edge Control Daemon v1.0.0-ea5 (api v1)
```

In order to mediate traffic to your clusters, Edge Control inserts itself into the DNS for your host (this is why it requires root access to run). It intercepts queries to your system’s primary DNS server, responds to queries that have to do with connected clusters, and forwards any other queries on to a fallback DNS server.

By default, the daemon intercepts queries to the primary DNS server listed in `/etc/resolv.conf`, and uses Google DNS on 8.8.8.8 or 8.8.4.4 for its fallback DNS server. You can override the choice of which DNS server to intercept using the `--dns` option, and you can override the fallback server using the `--fallback` option. For example, if `/etc/resolv.conf` is correct, but you have a local DNS server available on 10.0.0.1 that should be used for non-cluster queries, you could run

```bash
$ sudo edgectl daemon --fallback 10.0.0.1
Launching Edge Control Daemon v1.0.0-ea5 (api v1)
```

It's important that the primary DNS server and the fallback server be different. Otherwise Edge Control would forward queries to itself, resulting in a DNS loop.

3. Make sure everything is okay:

```bash
$ edgectl version
Client v1.0.0-ea5 (api v1)
Daemon v1.0.0-ea5 (api v1)

$ edgectl status
Not connected
```

The daemon’s logging output may be found in `/tmp/edgectl.log`.

### Upgrade

Tell the running daemon to exit with:

```bash
$ edgectl quit
Edge Control Daemon quitting...
```

Now you can grab the latest binary and launch the daemon again as above.

## Install Edge Control: Cluster

Depending on the type of cluster, your operations team may be involved. If you own the cluster, you will likely complete this setup yourself. If the cluster is shared, you may not have permission to complete these next steps, as the cluster owner will need to complete them.

### Traffic Manager

1. Install the Traffic Manager Kubernetes Deployment and Service using `kubectl`.
2. Fill in the name of the AES image before applying these manifests.
3. Note that the Traffic Manager needs access to your Ambassador Edge Stack license key, 
   so it needs to be installed in the same namespace. The manifests below assume that
   Ambassador Edge Stack is installed in the `ambassador` namespace.
4. Note also that the Traffic Manager must currently run as the `root` user.
5. Save these manifests in a YAML file:

```yaml
# This is traffic-manager.yaml
---
apiVersion: v1
kind: Service
metadata:
  name: telepresence-proxy
  namespace: ambassador
spec:
  type: ClusterIP
  clusterIP: None
  selector:
    app: telepresence-proxy
  ports:
    - name: sshd
      protocol: TCP
      port: 8022
    - name: api
      protocol: TCP
      port: 8081
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: telepresence-proxy
  namespace: ambassador
  labels:
    app: telepresence-proxy
spec:
  replicas: 1
  selector:
    matchLabels:
      app: telepresence-proxy
  template:
    metadata:
      labels:
        app: telepresence-proxy
    spec:
      containers:
      - name: telepresence-proxy
        image: __AES_IMAGE__   # Replace this
        command: [ "traffic-manager" ]
        ports:
          - name: sshd
            containerPort: 8022
        env:
          - name: AMBASSADOR_LICENSE_FILE
            value: /.config/ambassador/license-key
        volumeMounts:
          - mountPath: /tmp/ambassador-pod-info
            name: ambassador-pod-info
          - mountPath: /.config/ambassador
            name: ambassador-edge-stack-secrets
            readOnly: true
      restartPolicy: Always
      terminationGracePeriodSeconds: 0
      volumes:
      - downwardAPI:
          items:
          - fieldRef:
              fieldPath: metadata.labels
            path: labels
        name: ambassador-pod-info
      - name: ambassador-edge-stack-secrets
        secret:
          secretName: ambassador-edge-stack
``` 

5. Apply them:

```bash
$ kubectl apply -f traffic-manager.yaml
service/telepresence-proxy created
deployment.apps/telepresence-proxy created
```

### Traffic Agent

Any microservice running in a cluster with a traffic manager can opt in to intercept functionality by including the traffic agent in its pods. 

1. Since the Traffic Agent is built on Ambassador Edge Stack, it needs the same RBAC permissions that Ambassador does. The easiest way to provide this is to create a `ServiceAccount` in your service's namespace, bound to the `ambassador` `ClusterRole`:

```yaml
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: traffic-agent
  namespace: default
  labels:
    product: aes
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: traffic-agent
  labels:
    product: aes
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: ambassador
subjects:
- kind: ServiceAccount
  name: traffic-agent
  namespace: default
```

(If you want to include the Traffic Agent with multiple services, they can all use the same `ServiceAccount`.)

Copy the above YAML into `traffic-agent-rbac.yaml` and, if necessary, edit the two `namespace`s appropriately. Apply it:

```bash
$ kubectl apply -f traffic-agent-rbac.yaml
serviceaccount/traffic-agent created
clusterrolebinding.rbac.authorization.k8s.io/traffic-agent created
```

2. Next, you'll need to modify the YAML for each microservice to include the Traffic Agent. We'll start with a set of manifests for a simple microservice:

```yaml
# This is hello.yaml
---
apiVersion: v1
kind: Service
metadata:
  name: hello
  labels:
    app: hello
spec:
  selector:
    app: hello
  ports:
    - protocol: TCP
      port: 80
      targetPort: 8000              # Application port
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hello
  labels:
    app: hello
spec:
  replicas: 1
  selector:
    matchLabels:
      app: hello
  template:
    metadata:
      labels:
        app: hello
    spec:
      containers:                   # Application container
        - name: hello
          image: datawire/hello-world:latest
          ports:
            - containerPort: 8000   # Application port
```

In order to run the sidecar:

- you need to include the Traffic Agent container in the microservice pod;
- you need to switch the microservice's `Service` definition to point to the Traffic Agent's listening port (currently 8080 or 8443); and
- you need to tell the Traffic Agent how to set up for the microservice, using environment variables.

Here is a modified set of manifests that includes the Traffic Agent (with notes below):

```yaml
# This is hello-intercept.yaml
---
apiVersion: v1
kind: Service
metadata:
  name: hello
  labels:
    app: hello
spec:
  selector:
    app: hello
  ports:
    - protocol: TCP
      port: 80
      targetPort: 8080              # Traffic Agent port (note 1)
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hello
  labels:
    app: hello
spec:
  replicas: 1
  selector:
    matchLabels:
      app: hello
  template:
    metadata:
      labels:
        app: hello
    spec:
      containers:
        - name: hello               # Application container (note 2)
          image: datawire/hello-world:latest
          ports:
            - containerPort: 8000   # Application port
        - name: traffic-agent       # Traffic Agent container (note 3)
          image: __AES_IMAGE__      # Replace this (note 4)
          ports:
            - containerPort: 8080   # Traffic Agent port
          env:
          - name: AGENT_SERVICE     # Name to use for intercepting (note 5)
            value: hello
          - name: AGENT_PORT        # Port on which to talk to the microservice (note 6)
            value: "8000"
          - name: AGENT_MANAGER_NAMESPACE # Namespace for contacting the Traffic Manager (note 7)
            value: ambassador
          - name: AMBASSADOR_NAMESPACE # Namespace in which this microservice is running (note 8)
            valueFrom:
              fieldRef:
                fieldPath: metadata.namespace
      serviceAccountName: traffic-agent
```

Key points include:

- **Note 1**: The `Service` now points to the Traffic Agent’s port (8080) instead of the application’s port (8000).
- **Note 2**: The microservice's application container is actually unchanged.
- **Note 3**: The Traffic Agent's container has been added.
- **Note 4**: The Traffic Agent is included in the AES image. You'll need to edit this to have the actual image name.
- **Note 5**: The `AGENT_SERVICE` environment variable is mandatory. It sets the name that the Traffic Agent will report to the Traffic Manager for this microservice: you will have to provide this name to intercept this microservice.
- **Note 6**: The `AGENT_PORT` environment variable is mandatory. It tells the Traffic Agent the local port on which the microservice is listening.
- **Note 7**: The `AGENT_MANAGER_NAMESPACE` environment variable tells the Traffic Agent the namespace in which it will be able to find the Traffic Manager. If not present, it defaults to the `ambassador` namespace.
- **Note 8**: The `AMBASSADOR_NAMESPACE` environment variable is mandatory. It lets the Traffic Agent tell the Traffic Manager the namespace in which the microservice is running. 

In the future we will offer a tool to automate injecting the traffic agent into an existing microservice.

## Usage: Outbound Services

1. Starting with an empty cluster, add the simple microservice from above.

```bash
$ kubectl get svc,deploy
NAME                 TYPE        CLUSTER-IP   EXTERNAL-IP   PORT(S)   AGE
service/kubernetes   ClusterIP   10.43.0.1    <none>        443/TCP   27s

$ kubectl apply -f hello.yaml
service/hello created
deployment.apps/hello created

$ kubectl get svc,deploy
NAME                 TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)   AGE
service/hello        ClusterIP   10.43.111.189   <none>        80/TCP    7s
service/kubernetes   ClusterIP   10.43.0.1       <none>        443/TCP   2m12s

NAME                          READY   UP-TO-DATE   AVAILABLE   AGE
deployment.extensions/hello   0/1     1            0           7s
```

2. Use Edge Control to set up outbound connectivity to your cluster.

```bash
$ edgectl status
Not connected

$ edgectl connect
Connecting...
Connected to context default (https://localhost:6443)

Unable to connect to the traffic manager in your cluster.
The intercept feature will not be available.
Error was: kubectl get svc/deploy telepresency-proxy: exit status 1

$ edgectl status
Connected
  Context:       default (https://localhost:6443)
  Proxy:         ON (networking to the cluster is enabled)
  Intercepts:    Unavailable: no traffic manager

$ curl -L hello
Hello, world!
```

You are now able to connect to services directly from your laptop, as demonstrated by the `curl` command above.

3. When you’re done working with this cluster, disconnect.

```bash
$ edgectl disconnect
Disconnected

$ edgectl status
Not connected
```

## Usage: Intercept

1. Install the traffic manager in your cluster and the traffic agent in the simple microservice as described above.

```bash
$ kubectl apply -f traffic-manager.yaml
service/telepresence-proxy created
deployment.apps/telepresence-proxy created

$ kubectl apply -f hello-intercept.yaml
service/hello configured
deployment.apps/hello configured
```

2. Launch a local service on your laptop. If you were debugging the hello service, you might run a local copy in your debugger. In this example, we will start an arbitrary service on port 9000.

```bash
$ # using Python

$ python3 -m http.server 9000
Serving HTTP on 0.0.0.0 port 9000 (http://0.0.0.0:9000/) ...
[...]

$ # using NodeJS

$ npx http-server -p 9000
npx: installed 27 in 1.907s
Starting up http-server, serving ./
Available on:
  http://127.0.0.1:9000
  http://10.213.69.250:9000
Hit CTRL-C to stop the server
[...]
```

3. Connect to the cluster to set up outbound connectivity and check that you can access the hello service in the cluster with `curl`.

```bash
$ edgectl connect
Connecting...
Connected to context default (https://localhost:6443)

$ edgectl status
Connected
  Context:       default (https://localhost:6443)
  Proxy:         ON (networking to the cluster is enabled)
  Interceptable: 1 deployments
  Intercepts:    0 total, 0 local

$ curl -L hello
Hello, world!
```

4. Set up an intercept. In this example, we’ll capture requests that have the x-dev header set to $USER.

```bash
$ edgectl intercept avail
Found 1 interceptable deployment(s):
    1. hello

$ edgectl intercept list
No intercepts

$ edgectl intercept add hello -n example -m x-dev=$USER -t localhost:9000
Added intercept "example"

$ edgectl intercept list
1. example
    Intercepting requests to hello when
    - x-dev: ark3
    and redirecting them to localhost:9000

$ curl -L hello
Hello, world!

$ curl -L -H x-dev:$USER hello
<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01//EN" "http://www.w3.org/TR/html4/strict.dtd">
<html>
<head>
<meta http-equiv="Content-Type" content="text/html; charset=utf-8">
<title>Directory listing for /</title>
</head>
<body>
<h1>Directory listing for /</h1>
<hr>
<ul>
</ul>
<hr>
</body>
</html>
```

As you can see, the second request, which includes the specified x-dev header, is served by the local server.

5. Next, remove the intercept to restore normal operation.

```bash
$ edgectl intercept remove example
Removed intercept "example"

$ curl -L -H x-dev:$USER hello
Hello, world!
```

Requests are no longer intercepted.

## What's Next?

Multiple intercepts of the same deployment can run at the same time too. You can direct them to the same machine, allowing you to “or” together intercept conditions. Also, multiple developers can intercept the same deployment simultaneously. As long as their match patterns don’t collide, they don’t need to worry about disrupting one another.
