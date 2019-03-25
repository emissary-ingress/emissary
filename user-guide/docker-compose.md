# Deploying Ambassador to Docker Compose for local development

Docker Compose is useful for local development where Minikube may be undesirable. This guide is not intended for production deployments but it is intended to allow developers to quickly try out Ambassador features in a simple, local environment.

*It is important to note that any change to Ambassador's configuration using this method requires a restart of the Ambassador container and thus downtime.*

## Prerequisites

We assume that you have the latest version of Docker at the time of the writing of this guide.

## 1. Creating a simple Docker Compose environment

In this guide we will begin with a basic Ambassador API Gateway and add features over time. Not all features will be covered but by the end of this read you should know how to configure Ambassador to meet your local development needs.

### Create docker-compose.yaml file

In a working directory create a file called `docker-compose.yaml` with this content:

```yaml
version: '3.5'

services:
  ambassador:
    image: quay.io/datawire/ambassador:0.50.1
    ports:
    # expose port 80 via 8080 on the host machine
    - 8080:80
    volumes:
    # mount a volume where we can inject configuration files
    - ./config:/ambassador/ambassador-config
    environment:
    # don't try to watch Kubernetes for configuration changes
    - AMBASSADOR_NO_KUBEWATCH=no_kubewatch
```

Note the mounted volume. When Ambassador bootstraps on container startup it checks the `/ambassador/ambassador-config` directory for configuration files. We will use this behavior to configure ambassador.

Note also the `AMBASSADOR_NO_KUBEWATCH` environment variable. Without this, Ambassador will try to use the Kubernetes API to watch for service changes, which won't work in Docker.

### Create the initial configuration

Ambassador will interpret a total absence of configuration information as meaning that it should wait for dynamic configuration, so we'll give it a bare-bones configuration to get started.

Create a `config` folder (which must match the mounted volume in the `docker-compose.yaml` file) and add a file called `ambassador.yaml` to the directory.
(Note: Configuration files can have any name or combined into the same yaml file)

```bash
mkdir config
touch config/ambassador.yaml
```

Set the contents of the `config/ambassador.yaml` to this yaml configuration:

```yaml
---
apiVersion: ambassador/v1
kind: Module
name: ambassador
config: {}
```

This will allow Ambassador to come up with a completely default configuration.

### Test using Ambassador's Diagnostics

Run your compose environment and curl the diagnostics endpoint to ensure the compose file is working as expected.

```bash
# start your containers in the background
docker-compose up -d

# curl for the response header from the diagnostics endpoint
curl -I localhost:8080/ambassador/v0/diag/

# the response code should be 200
HTTP/1.1 200 OK
server: envoy
date: Fri, 17 Aug 2018 21:07:37 GMT
content-type: text/html; charset=utf-8
content-length: 6459
x-envoy-upstream-service-time: 10
```

## 2. Make a change to the default configuration

Let's turn off the diagnostics page to demonstrate how we will enable and configure Ambassador.

Edit the contents of the `config/ambassador.yaml` to this yaml configuration:

```yaml
---
apiVersion: ambassador/v1
kind: Module
name: ambassador
config:
  diagnostics:
    # Stop the diagnostics endpoint from being publicly available
    enabled: false
```

Now restart ambassador and test the diagnostics endpoint to ensure our configuration is in use:

```bash
# restart the container to pick up new configuration settings
docker-compose up -d -V ambassador

# curl the same diagnostics endpoint as the previous step
curl -I localhost:8080/ambassador/v0/diag/

# the response code should be 404
HTTP/1.1 404 Not Found
date: Fri, 17 Aug 2018 21:18:25 GMT
server: envoy
content-length: 0
```

Feel free to re-enable the diagnostics endpoint.

## 3. Add a route mapping

Now that we have demonstrated that we can modify the configuration let's add a mapping to route to `http://httpbin.org/` service.

Create a new file `config/mapping-httpbin.yaml` with these contents:

```yaml
---
apiVersion: ambassador/v1
kind:  Mapping
name:  httpbin_mapping
prefix: /httpbin/
service: httpbin.org
host_rewrite: httpbin.org   
```

Once again, restart ambassador and test the new mapping:

```bash
# restart the container to pick up new configuration settings
docker-compose up -d -V ambassador

# curl the quote-of-the-moment service
curl localhost:8080/qotm/quote/1

# the response body should be a json object with a quote
{
  "hostname": "qotm-3716059461-47tnl",
  "ok": true,
  "quote": "The light at the end of the tunnel is interdependent on the relatedness of motivation, subcultures, and management.",
  "time": "2018-08-17T21:29:24.690950",
  "version": "1.3"
}
```

## 3. Add a route mapping to an internal service

While routing to an external service is useful, more often than not our Docker Compose environment will contain a number of services that need routing.

### Add the qotm service to the docker-compose.yaml file

Edit the `docker-compose.yaml` file and add a new `qotm` service. It should now look like this:

```yaml
version: '3.5'

services:
  ambassador:
    image: quay.io/datawire/ambassador:0.50.1
    ports:
    - 8080:80
    volumes:
    # mount a volume where we can inject configuration files
    - ./config:/ambassador/ambassador-config
    environment:
    # don't try to watch Kubernetes for configuration changes
    - AMBASSADOR_NO_KUBEWATCH=no_kubewatch
  qotm:
    image: datawire/qotm:1.3
    ports:
    - 5000
```

### Update the mapping-qotm.yaml file to route to our internal qotm service

Edit the `config/mapping-qotm.yaml` file and modify the `service` and `rewrite` field. It should now look like this:

```yaml
---
apiVersion: ambassador/v1
kind:  Mapping
name:  qotm_mapping
prefix: /qotm/
# remove the qotm prefix when talking to the qotm service
rewrite: /
# change the `service` parameter to the name of our service with the port
service: qotm:5000
```

### Restart Ambassador and test

Re-run the same test as in the previous section to ensure the route works as before. This time we will need to bring up the new service first.

```bash
# start all new containers (eg. qotm)
docker-compose up -d

# restart the container to pick up new configuration settings
docker-compose up -d -V ambassador

# curl the quote-of-the-moment service
curl localhost:8080/qotm/quote/1

# the response body should be a json object with a quote
{
  "hostname": "qotm-3716059461-47tnl",
  "ok": true,
  "quote": "The light at the end of the tunnel is interdependent on the relatedness of motivation, subcultures, and management.",
  "time": "2018-08-17T21:29:24.690950",
  "version": "1.3"
}
```

### Verify that the local qotm container is serving requests

To ensure that the routing changes worked inspect the docker-compose logs to see the output from the local qotm service. It should look something like this:

```bash
> docker-compose logs
Attaching to ambassador-compose_qotm_1
qotm_1        | 2018-08-17 21:53:13 QotM 1.3 INFO: initializing on local-qotm:5000
qotm_1        | 2018-08-17 21:53:13 QotM 1.3 INFO:  * Running on http://0.0.0.0:5000/ (Press CTRL+C to quit)
qotm_1        | 2018-08-17 21:53:13 QotM 1.3 INFO:  * Restarting with stat
qotm_1        | 2018-08-17 21:53:14 QotM 1.3 INFO: initializing on local-qotm:5000
qotm_1        | 2018-08-17 21:53:14 QotM 1.3 WARNING:  * Debugger is active!
qotm_1        | 2018-08-17 21:53:14 QotM 1.3 INFO:  * Debugger PIN: 336-275-311
qotm_1        | 2018-08-17 21:53:41 QotM 1.3 DEBUG: GET /: session None, username None, handler statement
qotm_1        | 2018-08-17 21:53:41 QotM 1.3 INFO: 172.19.0.3 - - [17/Aug/2018 21:53:41] "GET / HTTP/1.1" 200 -
```

## 4. Add Authentication

The authentication module can be used to verify the identity and other security concerns at the entrypoint to the docker-compose cluster.

We will use the `datawire/ambassador-auth-service` container as an example.

### Create docker-compose.yaml service entry

Update the `docker-compose.yaml` file to include the new `auth` service:

```yaml
version: '3.5'

services:
  ambassador:
    image: quay.io/datawire/ambassador:0.50.1
    ports:
    - 8080:80
    volumes:
    # mount a volume where we can inject configuration files
    - ./config:/ambassador/ambassador-config
    environment:
    # don't try to watch Kubernetes for configuration changes
    - AMBASSADOR_NO_KUBEWATCH=no_kubewatch
  qotm:
    image: datawire/qotm:1.3
    ports:
    - 5000
  auth:
    image: datawire/ambassador-auth-service:latest
    ports:
    - 3000
```

### Create the auth.yaml configuration

Make a new file called `config/auth.yaml` with an auth definition inside:

```yaml
---
apiVersion: ambassador/v1
kind:  AuthService
name:  authentication
auth_service: "auth:3000"
path_prefix: "/extauth"
allowed_headers:
- "x-qotm-session"
```

This configuration will use the `AuthService` object to ensure that all requests made to ambassador are first sent to the `auth` docker container on port `3000` before being routed to the service that is mapped to the desired route. See the Authentication documentation for more details.

### Verify that Authentication is working

This sample authentication service only supports basic auth on a specific route. While the route is hardcoded you can implement your own that covers all routes. We will demonstrate that accessing the authenticated route with an incorrect Authorization header will result in a 401.

```bash
# start all new containers (eg. auth)
docker-compose up -d

# restart the api gateway to pick up new configuration settings
docker-compose up -d -V ambassador

# curl the quote-of-the-moment service without an auth header
curl -I localhost:8080/qotm/quote/1

# the response should look like this
HTTP/1.1 401 Unauthorized
x-powered-by: Express
x-request-id: da059f3e-7b9e-4d98-8428-f4f8ca742af7
www-authenticate: Basic realm="Ambassador Realm"
content-type: text/html; charset=utf-8
content-length: 0
etag: W/"0-2jmj7l5rSw0yVb/vlWAYkK/YBwk"
date: Fri, 17 Aug 2018 22:25:38 GMT
x-envoy-upstream-service-time: 1
server: envoy

# now try with a specific username and password
curl -I --user username:password localhost:8080/qotm/quote/1

# the response should be a 200
HTTP/1.1 200 OK
content-type: application/json
x-qotm-session: 5b75f31f-1155-4827-ab28-74d0f98573da
content-length: 217
server: envoy
date: Fri, 17 Aug 2018 22:29:43 GMT
x-envoy-upstream-service-time: 2
```

## 5. Tracing

As a final example we will configure Ambassador to send Zipkin traces to Jaeger. Integrating Zipkin into your services can be a vital glimpse into the performance bottlenecks of a distributed system.

### Add the Jaeger container to the docker-compose.yaml file

Building on our original `docker-compose.yaml` file, we can add a new service called `tracing` to the list:

```yaml
version: '3.5'

services:
  ambassador:
    image: quay.io/datawire/ambassador:0.50.1
    ports:
    - 8080:80
    volumes:
    # mount a volume where we can inject configuration files
    - ./config:/ambassador/ambassador-config
    environment:
    # don't try to watch Kubernetes for configuration changes
    - AMBASSADOR_NO_KUBEWATCH=no_kubewatch
  qotm:
    image: datawire/qotm:1.3
    ports:
    - 5000
  auth:
    image: datawire/ambassador-auth-service:latest
    ports:
    - 3000
  tracing:
    image: jaegertracing/all-in-one:latest
    environment:
      COLLECTOR_ZIPKIN_HTTP_PORT: 9411
    ports: 
      - 5775:5775/udp
      - 6831:6831/udp
      - 6832:6832/udp
      - 5778:5778
      - 16686:16686
      - 14268:14268
      - 9411:9411
```

### Create a tracing configuration file for Ambassador

Add a new configuration file `config/tracing.yaml` with these contents:

```yaml
---
apiVersion: ambassador/v1
kind: TracingService
name: tracing
service: tracing:9411
driver: zipkin
```

This will forward all of Ambassador's traces to the `tracing` service.

### Make requests and observe the traces

After reloading the Docker containers and configuration we should be able to make requests to the qotm service and see the traces in the Jaeger front-end UI.

```bash
# start all new containers (eg. tracing)
docker-compose up -d

# restart the api gateway to pick up new configuration settings
docker-compose up -d -V ambassador

# curl the quote-of-the-moment service as many times as you would like
curl --user username:password localhost:8080/qotm/quote/1
```

In a browser you can go to [http://localhost:16686/](http://localhost:16686/) and search for traces. To make this demonstration more useful one should implement Zipkin tracing middleware into their webserver.

## Next Steps

We have demonstrated that all the configurations that would normally be stored in kubernetes annotations can be saved as a yaml document in a volume mapped to `/ambassador/ambassador-config` within the Ambassador docker container. Hopefully this guide can be used to test new configurations locally before moving to a Kubernetes cluster. Of course, there will be differences between docker-compose and the Kubernetes implementation and one should be sure to test thoroughly in the latter before moving to production.
