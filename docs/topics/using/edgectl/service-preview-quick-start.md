# Service Preview in Action

When Service Preview is used, incoming requests get routed by Ambassador to a Traffic Agent, which then routes traffic to the microservice. When a request meets a specific criteria (e.g., it has a specific HTTP header value), the Traffic Agent will route that request to the microservice running locally. The following video shows Service Preview in more detail:

<iframe style="display: block; margin: auto;" width="560" height="315" src="https://www.youtube.com/embed/LDiyKOa1V_A" frameborder="0" allow="accelerometer; autoplay; encrypted-media; gyroscope; picture-in-picture" allowfullscreen></iframe>

## Quick Start

Service Preview creates a connection between your local environment and the cluster. These connections are managed through the Traffic Manager, which is deployed in your cluster, and the `edgectl` daemon, which runs in your local environment.

To get started with Service Preview, you'll need to [download and install the `edgectl` client](../edge-control#installing-edge-control).

> If you are a new user, or you are looking to start using Ambassador Edge Stack with Service Preview on a fresh installation, the `edgectl install` command will get you up and running in no time with a pre-configured Traffic Manager and Traffic Agent supported by automatic sidecar injection. You may also refer to [Introduction to Service Preview and Edge Control](../#installing-and-configuring-service-preview) for detailed instructions to manually install the Traffic Manager and configure a Traffic Agent alongside an existing Ambassador Edge Stack installation.

### Establishing a Connection with a Remote Cluster

There are three basic commands that are used for Service Preview:

1. Launch the edgectl daemon:

```bash
$ sudo edgectl daemon
Launching Edge Control Daemon v1.6.1 (api v1)
```

2. Connect your laptop to the cluster. This will enable your local environment to initiate traffic to the cluster.

```bash
$ edgectl connect
Connecting to traffic manager in namespace ambassador...
Connected to context k3s-default (https://172.20.0.3:6443)
```

3. Set up an intercept rule. This will enable the cluster initiate traffic to your local environment.

```bash
$ edgectl intercept add hello -n example -m x-dev=jane -t localhost:9000
```

### Usage: Outbound Services

1. Starting with an empty cluster, add the simple microservice from the [Introduction to Service Preview and Edge Control](../../edgectl#traffic-agent).

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

### Usage: Intercept

1. Install the traffic manager in your cluster and the traffic agent in the simple microservice as described in the [Introduction to Service Preview and Edge Control](../../edgectl#installing-and-configuring-service-preview).

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

6. Now let's set up an intercept with a preview URL.

Make sure your Host resource has preview URLs enabled.

```bash
$ kubectl get host minimal-host -o yaml
apiVersion: getambassador.io/v2
kind: Host
metadata:
  # [...]
spec:
  # [...]
  previewUrl:
    enabled: true
```

When you first edit your Host to enable preview URLs, you must reconnect to the cluster for the Edge Control Daemon to detect the change. This limitation will be removed in the future.

Now add an intercept and give it a try.

```bash
$ edgectl intercept avail
Found 1 interceptable deployment(s):
    1. hello

$ edgectl intercept list
No intercepts

$ edgectl intercept add hello -n example-url -t 9000
Using deployment hello in namespace default
Added intercept "example-url"
Share a preview of your changes with anyone by visiting
   https://staging.example.com/.ambassador/service-preview/0efb6d52-9ddc-410d-8717-8db58bac2088/

$ edgectl intercept list
   1. example-url
      (preview URL available)
      Intercepting requests to hello when
      - x-service-preview: 0efb6d52-9ddc-410d-8717-8db58bac2088
      and redirecting them to 127.0.0.1:9000
Share a preview of your changes with anyone by visiting
   https://staging.example.com/.ambassador/service-preview/0efb6d52-9ddc-410d-8717-8db58bac2088/

$ curl https://staging.example.com/hello/
Hello, world!

$ curl https://staging.example.com/.ambassador/service-preview/0efb6d52-9ddc-410d-8717-8db58bac2088/hello/
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

As you can see, the second request, which uses the preview URL, is served by the local server.

7. Next, remove the intercept to restore normal operation.

```bash
$ edgectl intercept remove example-url
Removed intercept "example-url"

$ curl https://staging.example.com/.ambassador/service-preview/0efb6d52-9ddc-410d-8717-8db58bac2088/hello/
Hello, world!
```

Requests are no longer intercepted.

## What's Next?

Multiple intercepts of the same deployment can run at the same time too. You can direct them to the same machine, allowing you to “or” together intercept conditions. Also, multiple developers can intercept the same deployment simultaneously. As long as their match patterns don’t collide, they don’t need to worry about disrupting one another.
