# Edge Control

Edge Control is the command-line tool for installing and managing the Ambassador Edge Stack. And Edge Control's outbound and intercept features allow developers to preview changes to their services while sharing a single development cluster.

If you are a developer working on a service that depends on other in-cluster services, use `edgectl connect` to set up connectivity from your laptop to the cluster. This allows software on your laptop, such as your work-in-progress service running in your debugger, to connect to other services in the cluster.

When you want to test your service with traffic from the cluster, use `edgectl intercept` to designate a subset of requests for this service to be redirected to your laptop. You can use those requests to test and debug your local copy of the service. All other requests will go to the existing service running in the cluster without disruption.

## Installing Edge Control

Edge Control is available as a downloadable executable for both Mac OS X and Linux. While Edge Control clients are available for Windows, these binaries do not support Service Preview.

For MacOS:

```bash
curl -fLO https://metriton.datawire.io/downloads/darwin/edgectl
chmod a+x edgectl
xattr -d com.apple.quarantine edgectl # Give OS X permission to run the executable
mv edgectl ~/bin  # Somewhere in your PATH
```

For Linux:

```bash
curl -fLO https://metriton.datawire.io/downloads/linux/edgectl
chmod a+x edgectl
mv edgectl ~/bin  # Somewhere in your PATH
```

### Upgrading

Make sure you've terminated the daemon.

```bash
$ edgectl quit
Edge Control Daemon quitting...
```

Download the latest binary, as above, and replace your existing binary.

## Service Preview Quick Start

Service Preview creates a connection between your local environment and the cluster. These connections are managed through the Traffic Manager, which is deployed in your cluster, and the `edgectl` daemon, which runs in your local environment.

There are three basic commands that are used for Service Preview:

1. Launch the edgectl daemon:

```bash
$ sudo edgectl daemon
Launching Edge Control Daemon v1.3.2 (api v1)
```

2. Connect your laptop to the cluster. This will enable your local environment to initiate traffic to the cluster.

```
$ edgectl connect
Connecting to traffic manager in namespace ambassador...
Connected to context k3s-default (https://172.20.0.3:6443)
```

3. Set up an intercept rule. This will enable the cluster initiate traffic to your local environment.

```
$ edgectl intercept add hello -n example -m x-dev=jane -t localhost:9000

```


## Edge Control commands

### `edgectl connect`

Connect to the cluster. This command allows your local environment to initiate traffic to the cluster, allowing services running locally to send and receive requests to cluster services.

```
$ edgectl connect
Connecting to traffic manager in namespace ambassador...
Connected to context gke_us-east1-b_demo-cluster (https://35.136.57.145)
```

### `edgectl daemon`

In order to mediate traffic to your clusters, Edge Control inserts itself into the DNS for your host (this is why it requires root access to run). It intercepts queries to your system’s primary DNS server, responds to queries that have to do with connected clusters, and forwards any other queries on to a fallback DNS server.

By default, the daemon intercepts queries to the primary DNS server listed in `/etc/resolv.conf`, and uses Google DNS on 8.8.8.8 or 8.8.4.4 for its fallback DNS server. You can override the choice of which DNS server to intercept using the `--dns` option, and you can override the fallback server using the `--fallback` option.

It's important that the primary DNS server and the fallback server be different. Otherwise Edge Control would forward queries to itself, resulting in a DNS loop.

The daemon’s logging output may be found in `/tmp/edgectl.log`.

#### Examples

Launch Daemon:

```bash
$ sudo edgectl daemon
Launching Edge Control Daemon v1.0.0-ea5 (api v1)
```

If `/etc/resolv.conf` is correct, but you have a local DNS server available on 10.0.0.1 that should be used for non-cluster queries, you could run Configure fallback server:

```bash
$ sudo edgectl daemon --fallback 10.0.0.1
Launching Edge Control Daemon v1.0.0-ea5 (api v1)
```

### `edgectl disconnect`

Disconnect from the cluster.

### `edgectl intercept`

Intercept enables the cluster to initiate traffic to the local environment. To prevent unwanted traffic from being routed to the cluster, `intercept` creates routing rules that specify which traffic to send to the local environment. An `intercept` is created on a per (Kubernetes) deployment basis. Each deployment must have a traffic agent installed in order for `intercept` to function.

#### `edgectl intercept available`

List available Kubernetes deployments for intercept.

```
$ edgectl intercept available
Found 2 interceptable deployment(s):
   1. xyz in namespace default
   2. hello in namespace default
```

#### `edgectl intercept list`

List the current active intercepts.

#### `edgectl intercept add`

Add an intercept. The basic format of this command is:

```
  edgectl intercept add DEPLOYMENT -n NAME -t [HOST:]PORT -m HEADER=REGEX ...
```

* DEPLOYMENT specifies a Kubernetes deployment with a traffic agent installed. You can get the list of available deployments with the `intercept available` command.
* `--name` or `-n` specifies a name for an intercept.
* `--target` or `-t` specifies the target of an intercept. Typically, this is a service running in the local environment that is a virtual replacement for the deployment in the cluster.
* `--match` or `-m` specifies a match rule on requests. Requests that are sent to the traffic agent that match this rule will be routed to the target.

A few other options to `intercept` include:

* `--namespace` to specify the Kubernetes namespace in which to create a mapping for intercept
* `--prefix` or `-p` which specifies a prefix to intercept (the default is `/`)

#### Example

Intercept all requests to the `hello` deployment that match the HTTP `x-dev` header with a value of `jane` to a service running locally on port 9000:

```
$ edgectl intercept add hello -n example -m x-dev=jane -t localhost:9000
Added intercept "example"
```

### `edgectl pause`

Pause the daemon. The network overrides used by the edgectl daemon are temporarily disabled. Typically, this is used for connecting with a VPN that is not compatible with Edge Control.

```
$ edgectl pause
Network overrides paused.
Use "edgectl resume" to reestablish network overrides.
```

### `edgectl quit`

Quit the daemon. Ensure that the daemon has quit prior to upgrades.

### `edgectl resume`

Resume the daemon. Used after `edgectl pause`.

### `edgectl status`

Print the status of Edge Control, including the Kubernetes context that is currently being used.

```
$ edgectl status
Connected
  Context:       gke_us-east1-b_demo-cluster (https://35.136.57.145)
  Proxy:         ON (networking to the cluster is enabled)
  Interceptable: 2 deployments
  Intercepts:    0 total, 0 local
```

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
