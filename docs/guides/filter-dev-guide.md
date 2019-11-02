# Developing Filters

Sometimes you may want Ambassador Edge Stack to manipulate an incoming request. Some example use cases:

* Inspect an incoming request, and add a custom header that can then be used for routing
* Add custom Authorization headers
* Validate an incoming request fits an OpenAPI specification before passing the request to a target service


Ambassador Edge Stack supports these use cases by allowing you to execute custom logic in `Filters`. Filters are written in Golang, and managed by Ambassador Pro.



## Creating and Deploying Filters

We've created an example filter that you can customize for your particular use case.

1. Start with the example filter: `git clone
   https://github.com/datawire/apro-example-plugin/`.

2. Make code changes to `param-plugin.go`. Note: If you're developing a non-trivial filter, see the rapid development section below for a faster way to develop and test your filter.

3. Run `make DOCKER_REGISTRY=...`, setting `DOCKER_REGISTRY` to point
   to a registry you have access to. This will generate a Docker image
   named `$DOCKER_REGISTRY/amb-sidecar-plugin:VERSION`.

4. Push the image to your Docker registry: `docker push $DOCKER_REGISTRY/amb-sidecar-plugin:VERSION`.

5. Configure Ambassador Pro to use the plugin by creating a `Filter`
   and `FilterPolicy` CRD, as per the [filter reference](/reference/filter-reference).

6. Update the standard Ambassador Pro manifest to use your Docker
   image instead of the standard sidecar.

   ```patch
      containers:
      - name: ambassador-pro
   -    image: quay.io/datawire/ambassador_pro:amb-sidecar-%aproVersion%
   +    image: DOCKER_REGISTRY/amb-sidecar-plugin:VERSION
        ports:
        - name: ratelimit-grpc
          containerPort: 8081
        - name: ratelimit-debug
          containerPort: 6070
   ```

## Rapid development of a custom filter

During development, you may want to sidestep the deployment process for a faster development loop. The `apro-plugin-runner` helps you rapidly develop Ambassador Edge Stack filters locally.

To install the runner, download the latest version:

<a class="pro-runner-dl" href="https://s3.amazonaws.com/datawire-static-files/apro-plugin-runner/%aproVersion%/darwin/amd64/apro-plugin-runner">Mac 64-bit</a> |
<a class="pro-runner-linux-dl" href="https://s3.amazonaws.com/datawire-static-files/apro-plugin-runner/%aproVersion%/linux/amd64/apro-plugin-runner">Linux 64-bit</a>

Note that the plugin runner must match the version of Ambassador Pro that you are running. Place the binary somewhere in your `$PATH`.

Information about open source code used in `apro-plugin-runner` can be found by running `apro-plugin-runner --version`.

Now, you can quickly test and develop your filter.

1. In your filter directory, type: `apro-plugin-runner :8080 ./param-plugin.so`.
2. Test the filter by running `curl`:

    ```
    $ curl -v localhost:8080?db=2
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

## Further reading

For more details about configuring filters and the `plugin` interface, see the [filter reference](/reference/filter-reference).
