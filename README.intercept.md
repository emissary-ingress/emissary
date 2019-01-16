# Intercept

Machinery for `telepresence intercept`

## How it works

- The Telepresence Proxy runs as a persistent service in the cluster
- You run a sidecar next to your application and point your Kubernetes service at the Envoy running in the sidecar
- By default, that Envoy sends all requests to your application.
- When the user launches an intercept, Telepresence talks to the Proxy running in the cluster, which in turn talks to the sidecar.
- The sidecar reconfigures Envoy to send a subset of requests to a particular port on the Proxy.
- Telepresence uses SSH port forwarding to get those requests down from the Proxy to the local machine.

## Quick and dirty demo

Here's the demo I did over Zoom on Friday afternoon.

### The Application

The application is a [simple echo server](https://github.com/jmalloc/echo-server) for HTTP requests. It responds to requests on port 8080 with its hostname and the headers it received.

#### Local

I have the application running on my local machine in Docker with the port published.

```shell
$ docker run --rm -d -h container_on_laptop -p 8080:8080 jmalloc/echo-server
06c51f70a36037e3b94bfb413148129b8ac97fecdc4ebff96d0f4ea437496181

$ curl localhost:8080/foo/bar
Request served by container_on_laptop

HTTP/1.1 GET /foo/bar

Host: localhost:8080
User-Agent: curl/7.54.0
Accept: */*

$ 
```

#### Cluster

I have the application running in the cluster as the-app. The manifests I used to launch it are in `example/the-app.yaml`.

The application runs in the first container, listening on port 8080 as before. The second container runs the sidecar image, which includes Envoy listening on port 9900. The Kubernetes service points to that Envoy, which is configured to redirect everything to the app. It does this by obtaining the correct port number from the environment variable `APPPORT`. Without this variable set, the sidecar cannot do anything.

The sidecar's environment must also include `APPNAME`, which is the name you pass to Telepresence at the command line. We should probably make this the same as the deployment name by convention. Without this variable set, the sidecar is unable to offer intercept capability and just acts as a dumb forwarder. Maybe we can figure out how to compute this name directly, but the key is that every pod (replica) in the deployment needs to have the same name so they are all configured the same. In this example `APPNAME` is set to `myapp` for some reason.

Hitting the application in the cluster reveals slightly different behavior due to the presence of Envoy. I'm using a `telepresence outbound` session for this, but you can use a classic `telepresence --run curl ...` or a `kubectl exec` or whatever.

```shell
$ curl the-app/foo/bar
Request served by the-app-65c6ddc48d-8phxl

HTTP/1.1 GET /foo/bar

Host: the-app
Accept: */*
X-Forwarded-Proto: http
X-Request-Id: 6825e3c8-1dbb-44fe-bb6d-b11c4ca75ad0
X-Envoy-Expected-Rq-Timeout-Ms: 15000
Content-Length: 0
User-Agent: curl/7.54.0

$ 
```

You can see that Envoy has added a couple of headers going in.

### The Proxy

The Telepresence Proxy runs a small Go program to keep track of running intercept sessions and communicate with the sidecars. Each sidecar repeatedly long-polls the proxy; this allows the proxy to push updates to the sidecars immediately. The Proxy also runs a normal OpenSSH server so that Telepresence can establish tunnels in the usual way.

The manifests I used to launch the Proxy are in `example/proxy.yaml`. These are long and repetitive because they're exposing 16 ports for port forwards in addition to the API port and the SSHD port.

### The Demo

As things are set up, requests to the app always go to the pods.

```shell
$ curl the-app/foo/bar
Request served by the-app-65c6ddc48d-8phxl

HTTP/1.1 GET /foo/bar

Host: the-app
Content-Length: 0
User-Agent: curl/7.54.0
Accept: */*
X-Forwarded-Proto: http
X-Request-Id: 86dee54f-1c7e-4559-9abe-152c69170329
X-Envoy-Expected-Rq-Timeout-Ms: 15000

$ 

$ curl the-app/ark3
Request served by the-app-65c6ddc48d-8phxl

HTTP/1.1 GET /ark3

Host: the-app
X-Forwarded-Proto: http
X-Request-Id: 2c108337-f620-421f-9b8b-7d8e70b727df
X-Envoy-Expected-Rq-Timeout-Ms: 15000
Content-Length: 0
User-Agent: curl/7.54.0
Accept: */*

$ 
```

Now let's set up an intercept. You'll need the HEAD version of Telepresence. If you've already installed 0.94, then you can grab [this blob](https://3007-82933315-gh.circle-artifacts.com/0/home/circleci/project/dist/telepresence) from CircleCI's artifacts storage, set the executable bit, and use it instead. In my examples, `telepresence` is 0.94, which is required for `telepresence outbound`, and `teldev` is HEAD.

**Update:** The linked blob can run both `telepresence outbound` and `telepresence intercept`. I fixed the interactive `sudo` issue that was breaking `outbound` on head.

```shell
$ telepresence version
0.94

$ teldev version
0.94-26-g9290cab

$ teldev intercept myapp -m :path ".*ark3.*" --name test -p 8080
T: Setting up intercept session test
T: Intercepting requests to myapp
T: and redirecting them to localhost:8080
T: when the following headers match:
T:   :path: .*ark3.*
T: Connecting to the Telepresence Proxy
T: Intercept is running. Press Ctrl-C/Ctrl-Break to quit.
```

This intercept command sets up `myapp` for intercept. Requests where the value of the `:path` header (really the request path) matches the regular expression `.*ark3.*` (i.e. has `ark3` in it anywhere) will get directed to port 8080 on this computer. You can specify multiple `-m` options; they must *all* match for the request to be intercepted.

Let's try the same requests as before:

```shell
$ curl the-app/foo/bar
Request served by the-app-65c6ddc48d-8phxl

HTTP/1.1 GET /foo/bar

Host: the-app
User-Agent: curl/7.54.0
Accept: */*
X-Forwarded-Proto: http
X-Request-Id: aca2e59b-e4e2-4d91-99d2-14f151fb003a
X-Envoy-Expected-Rq-Timeout-Ms: 15000
Content-Length: 0

$ 
$ curl the-app/ark3
Request served by container_on_laptop

HTTP/1.1 GET /ark3

Host: the-app
User-Agent: curl/7.54.0
Accept: */*
X-Forwarded-Proto: http
X-Request-Id: be953be2-ba7f-4b17-aee6-d61f8837961f
X-Envoy-Expected-Rq-Timeout-Ms: 15000
Content-Length: 0

$ 
```

As you can see, the second request, which includes "ark3" in the path, is served by the container running on my laptop.

Now if we hit Ctrl-C on the intercept command:

```shell
[...]
T: Intercept is running. Press Ctrl-C/Ctrl-Break to quit.
^CT: Keyboard interrupt (Ctrl-C/Ctrl-Break) pressed
T: Exit cleanup in progress

$ 
```

Then the requests above go back to always going to the cluster.

```shell
$ curl the-app/foo/bar
Request served by the-app-65c6ddc48d-8phxl

HTTP/1.1 GET /foo/bar

Host: the-app
User-Agent: curl/7.54.0
Accept: */*
X-Forwarded-Proto: http
X-Request-Id: 5f832ff9-537a-406f-a27a-b23898f42abd
X-Envoy-Expected-Rq-Timeout-Ms: 15000
Content-Length: 0

$ 
$ curl the-app/ark3
Request served by the-app-65c6ddc48d-8phxl

HTTP/1.1 GET /ark3

Host: the-app
User-Agent: curl/7.54.0
Accept: */*
X-Forwarded-Proto: http
X-Request-Id: 48a29066-3caa-4282-b172-8bfd62234551
X-Envoy-Expected-Rq-Timeout-Ms: 15000
Content-Length: 0

$ 
```

Multiple intercepts of the same deployment can run at the same time too. You can direct them to the same machine, allowing you to "or" together intercept conditions. Or multiple developers can intercept the same deployment simultaneously. As long as their headers don't collide, they don't need to worry about disrupting one another.
