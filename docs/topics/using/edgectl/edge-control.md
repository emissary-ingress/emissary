> **Service Preview has been replaced by Telepresence, these docs will remain as a historical reference. [Learn more about Telepresence](../../../../telepresence/faqs) or [go to the quick start guide](../../../../telepresence/quick-start/).**

# Edge Control Reference

Edge Control is the command-line tool for installing and managing the Ambassador Edge Stack. And Edge Control's outbound and intercept features allow developers to preview changes to their services while sharing a single development cluster.

If you are a developer working on a service that depends on other in-cluster services, use `edgectl connect` to set up connectivity from your laptop to the cluster. This allows software on your laptop, such as your work-in-progress service running in your debugger, to connect to other services in the cluster.

When you want to test your service with traffic from the cluster, use `edgectl intercept` to designate a subset of requests for this service to be redirected to your laptop. You can use those requests to test and debug your local copy of the service. All other requests will go to the existing service running in the cluster without disruption.

## Installing Edge Control

Edge Control is available as a downloadable executable for both Mac OS X and Linux. While Edge Control clients are available for Windows, these binaries do not support Service Preview.

For MacOS:

```
sudo curl -fL https://metriton.datawire.io/downloads/darwin/edgectl \
  -o /usr/local/bin/edgectl && \
sudo chmod a+x /usr/local/bin/edgectl
```

For Linux:

```
sudo curl -fL https://metriton.datawire.io/downloads/linux/edgectl \
  -o /usr/local/bin/edgectl && \
sudo chmod a+x /usr/local/bin/edgectl
```

### Upgrading

Make sure you've terminated the daemon.

```
$ edgectl quit
Edge Control Daemon quitting...
```

Download the latest binary, as above, and replace your existing binary.

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

```
$ sudo edgectl daemon
Launching Edge Control Daemon v1.0.0-ea5 (api v1)
```

If `/etc/resolv.conf` is correct, but you have a local DNS server available on 10.0.0.1 that should be used for non-cluster queries, you could run Configure fallback server:

```
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
* `--grpc` to instruct Envoy to use HTTP/2 to communicate with the target deployment (the default is `false`)

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

## What's Next?

See how [Edge Control commands can be used in action](../service-preview-tutorial) to establish outbound connectivity with a remote Kubernetes cluster and intercept inbound requests.
