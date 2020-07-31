# Service Preview Tutorial

When Service Preview is used, incoming requests get routed by Ambassador to a Traffic Agent, which then routes traffic to the microservice. When a request meets a specific criteria (e.g., it has a specific HTTP header value), the Traffic Agent will route that request to the microservice running locally. The following video shows Service Preview in more detail:

<iframe style="display: block; margin: auto;" width="560" height="315" src="https://www.youtube.com/embed/LDiyKOa1V_A" frameborder="0" allow="accelerometer; autoplay; encrypted-media; gyroscope; picture-in-picture" allowfullscreen></iframe>

## Quick Start

Service Preview creates a connection between your local environment and the cluster. These connections are managed through the Traffic Manager, which is deployed in your cluster, and the `edgectl` daemon, which runs in your local environment.

To get started with Service Preview, you'll need to [download and install the `edgectl` client](../edge-control#installing-edge-control).

Service Preview should already by installed in your cluster before starting this quick start. See the [installation instructions](../service-preview-install) for information on how to install Service Preview.

### Intercepting Traffic

One of the main use cases of Service Preview is to intercept certain requests to services in your Kubernetes cluster and route them to your laptop instead.

#### Intercept with an HTTP header

1. Make sure sure that the `Hello` is installed. See the [installation instructions](../service-preview-install). 

   ```
   $ kubectl get svc,deploy
   NAME                 TYPE        CLUSTER-IP   EXTERNAL-IP   PORT(S)   AGE
   service/hello        ClusterIP   10.4.28.14   <none>        80/TCP    6m18s
   service/kubernetes   ClusterIP   10.4.16.1    <none>        443/TCP   25m
    
   NAME                          READY   UP-TO-DATE   AVAILABLE   AGE
   deployment.extensions/hello   1/1     1            1           6m18s
   ```

2. Launch a local service on your laptop. If you were debugging the `Hello` service, you might run a local copy in your debugger. In this example, we will start an arbitrary service on port 9000.

   ```
   # using Python
   $ python3 -m http.server 9000 &
   Serving HTTP on :: port 9000 (http://[::]:9000/) ...
   ```

3. Make sure you are connected to the cluster to set up outbound connectivity and check that you can access the `Hello` service in the cluster with `curl`.

   ```
   $ edgectl connect
   Already connected
    
   $ edgectl status
   Connected
     Context:       default (https://localhost:6443)
     Proxy:         ON (networking to the cluster is enabled)
     Interceptable: 1 deployments
     Intercepts:    0 total, 0 local
    
   $ curl -L hello
   Hello, world!
   ```

4. Set up an intercept. In this example, we’ll capture requests that have the `x-dev` header set to $USER.

   ```
   $ edgectl intercept avail
   Found 1 interceptable deployment(s):
      1. hello in namespace default
    
   $ edgectl intercept list
   No intercepts
    
   $ edgectl intercept add hello -n example -m x-dev=$USER -t localhost:9000
   Using deployment hello in namespace default
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

   As you can see, the second request, which includes the specified `x-dev` header, is served by the local server.

5. Next, remove the intercept to restore normal operation.

   ```
   $ edgectl intercept remove example
   Removed intercept "example"
    
   $ curl -L -H x-dev:$USER hello
   Hello, world!
   ```

   Requests are no longer intercepted.

#### Intercept with a Preview URL

Now let's set up an intercept with a preview URL.

1. Create or edit an existing `Host` resource to enable Preview URLs

   ```yaml
   ---
   apiVersion: getambassador.io/v2
   kind: Host
   metadata:
     name: preview-host
   spec:
     hostname: {{AMBASSADOR_IP_OR_DOMAIN_NAME}}
     # [...]
     previewUrl:
       enabled: true
       type: path
   ```

   Replace `{{AMBASSADOR_IP_OR_DOMAIN_NAME}}` with the IP address or domain name of your Ambassador service and apply it with `kubectl`

   ```
   kubectl apply -f preview-host
   ```

2. Refresh the edgectl connection for it to detect the new `Host`

   ```
   $ edgectl disconnect
   Disconnected
    
   $ edgectl connect
   Connecting to traffic manager in namespace ambassador...
   Connected to context k3s-default (https://172.20.0.3:6443)
   ```

3. Now add an intercept and give it a try.

   ```
   $ edgectl intercept avail
   Found 1 interceptable deployment(s):
       1. hello in namespace default
    
   $ edgectl intercept list
   No intercepts
    
   $ edgectl intercept add hello -n example-url -t 9000
   Using deployment hello in namespace default
   Added intercept "example-url"
   Share a preview of your changes with anyone by visiting
    https://staging.example.com/.ambassador/service-preview/251b550a-66e4-47f3-aa5e-97801b4037a8/
    
   $ edgectl intercept list
      1. example-url
         (preview URL available)
         Intercepting requests to hello when
         - x-service-preview: 251b550a-66e4-47f3-aa5e-97801b4037a8
         and redirecting them to 127.0.0.1:9000
   Share a preview of your changes with anyone by visiting
      https://staging.example.com/.ambassador/service-preview/251b550a-66e4-47f3-aa5e-97801b4037a8/
    
   $ curl https://staging.example.com/hello/
   Hello, world!
    
   $ curl https://staging.example.com/.ambassador/service-preview/251b550a-66e4-47f3-aa5e-97801b4037a8/hello/
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

4. Remove the intercept to restore normal operation.

  ```
  $ edgectl intercept remove example-url
  Removed intercept "example-url"
   
  $ curl https://staging.example.com/.ambassador/service-preview/0efb6d52-9ddc-410d-8717-8db58bac2088/hello/
  Hello, world!
  ```
  
  Requests are no longer intercepted.

### Outbound Services

Service Preview bridges your local and cluster DNS. This allows for the use case of using Service Preview as a debug tool for interacting with services in your cluster.

1. Make sure sure that the `Hello` service is installed. See the [installation instructions](../service-preview-install). 

   ```
   $ kubectl get svc,deploy
   NAME                 TYPE        CLUSTER-IP   EXTERNAL-IP   PORT(S)   AGE
   service/hello        ClusterIP   10.4.28.14   <none>        80/TCP    6m18s
   service/kubernetes   ClusterIP   10.4.16.1    <none>        443/TCP   25m
    
   NAME                          READY   UP-TO-DATE   AVAILABLE   AGE
   deployment.extensions/hello   1/1     1            1           6m18s
   ```

2. Make sure you are still connected to the cluster.

   ```
   $ edgectl connect
   Already connected
    
   $ edgectl status
   Connected
     Context:       default (https://34.72.18.227)
     Proxy:         ON (networking to the cluster is enabled)
     Interceptable: 1 deployments
     Intercepts:    0 total, 0 local
    
   $ curl -L hello
   Hello, world!
   ```

You are now able to connect to services directly from your laptop, as demonstrated by the `curl` command above.

3. When you’re done working with this cluster, disconnect.

   ```
   $ edgectl disconnect
   Disconnected
    
   $ edgectl status
   Not connected
   ```

## What's Next?

Multiple intercepts of the same deployment can run at the same time too. You can direct them to the same machine, allowing you to “or” together intercept conditions. Also, multiple developers can intercept the same deployment simultaneously. As long as their match patterns don’t collide, they don’t need to worry about disrupting one another.
