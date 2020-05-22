# Developing Custom Filters for Routing

Sometimes you may want Ambassador Edge Stack to manipulate an incoming request. Some example use cases:

* Inspect an incoming request, and add a custom header that can then be used for routing
* Add custom Authorization headers
* Validate an incoming request fits an OpenAPI specification before passing the request to a target service

Ambassador Edge Stack supports these use cases by allowing you to execute custom logic in `Filters`. Filters are written in Golang, and managed by Ambassador Edge Stack. If you want to write a filter in a language other than Golang, Ambassador also has an HTTP/gRPC interface for Filters called `External`.

## Prerequisites

`Plugin` `Filter`s are built as [Go plugins](https://golang.org/pkg/plugin/) and loaded directly into the Ambassador Edge Stack container so they can run in-process with the rest of Ambassador Edge Stack.

To build a `Plugin` `Filter` into the Ambassador Edge Stack container you will need
- Linux or MacOS host (Windows Subsystem for Linux is ok)
- [Docker](https://docs.docker.com/install/) 
- [rsync](https://rsync.samba.org/)

The `Plugin` `Filter` is built by `make` which uses Docker to create a stable build environment in a container and `rsync` to copy files between the container and your host machine.

See the [README](https://github.com/datawire/apro-example-plugin) for more information on how the `Plugin` works.

## Creating and Deploying Filters

We've created an example filter that you can customize for your particular use case.

1. Start with the example filter: `git clone
   https://github.com/datawire/apro-example-plugin/`.

2. Make code changes to `param-plugin.go`. Note: If you're developing a non-trivial filter, see the rapid development section below for a faster way to develop and test your filter.

3. Run `make DOCKER_REGISTRY=...`, setting `DOCKER_REGISTRY` to point
   to a registry you have access to. This will generate a Docker image
   named `$DOCKER_REGISTRY/amb-sidecar-plugin:VERSION`.

4. Push the image to your Docker registry: `docker push $DOCKER_REGISTRY/amb-sidecar-plugin:VERSION`.

5. Configure Ambassador Edge Stack to use the plugin by creating a `Filter`
   and `FilterPolicy` CRD, as per the [filter reference](../../topics/using/filters).

6. Update the standard Ambassador Edge Stack manifest to use your Docker
   image instead of the standard sidecar.

   ```patch
              value: https://127.0.0.1:8443
            - name: AMBASSADOR_ADMIN_URL
              value: http://127.0.0.1:8877
   -        image: docker.io/datawire/aes:$version$
   +        image: DOCKER_REGISTRY/aes-plugin:VERSION
            imagePullPolicy: Always
            livenessProbe:
              httpGet:
   ```

## Rapid Development of a Custom Filter

During development, you may want to sidestep the deployment process for a faster development loop. The `aes-plugin-runner` helps you rapidly develop Ambassador Edge Stack filters locally.

To install the runner, download the latest version:

<a class="pro-runner-dl" href="https://s3.amazonaws.com/datawire-static-files/aes-plugin-runner/$version$/darwin/amd64/aes-plugin-runner">Mac 64-bit</a> |
<a class="pro-runner-linux-dl" href="https://s3.amazonaws.com/datawire-static-files/aes-plugin-runner/$version$/linux/amd64/aes-plugin-runner">Linux 64-bit</a>

Note that the plugin runner must match the version of Ambassador Edge Stack that you are running. Place the binary somewhere in your `$PATH`.

Information about open-source code used in `aes-plugin-runner` can be found by running `aes-plugin-runner --version`.

Now, you can quickly test and develop your filter.

1. In your filter directory, type: `aes-plugin-runner :8080 ./param-plugin.so`.
2. Test the filter by running `curl`:

    ```
    $ curl -Lv localhost:8080?db=2
    * Rebuilt URL to: localhost:8080/?db=2
    *   Trying ::1...
    * TCP_NODELAY set
    * Connected to localhost (::1) port 8080 (#0)
    > GET /?db=2 HTTP/1.1
    > Host: localhost:8080
    > User-Agent: curl/7.54.0
    > Accept: */*
    >
    < HTTP/1.1 200 OK
    < X-Dc: Even
    < Date: Mon, 25 Feb 2019 19:58:38 GMT
    < Content-Length: 0
    <
    * Connection #0 to host localhost left intact
    ```

Note in the example above the `X-Dc` header is added. This lets you inspect the changes the filter is making to your HTTP header.

## Further Reading

For more details about configuring filters and the `plugin` interface, see the [filter reference](../../topics/using/filters/).
