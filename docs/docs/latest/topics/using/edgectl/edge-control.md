# Edge Control

Edge Control is the command-line tool for installing and managing the Ambassador Edge Stack. And Edge Control's outbound and intercept features allow developers to preview changes to their services while sharing a single development cluster.

If you are a developer working on a service that depends on other in-cluster services, use `edgectl connect` to set up connectivity from your laptop to the cluster. This allows software on your laptop, such as your work-in-progress service running in your debugger, to connect to other services in the cluster.

When you want to test your service with traffic from the cluster, use `edgectl intercept` to designate a subset of requests for this service to be redirected to your laptop. You can use those requests to test and debug your local copy of the service running. All other requests will go to the existing service running in the cluster without disruption.

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
