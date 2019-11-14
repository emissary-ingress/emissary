# Ambassador Open Source Software (OSS)

In this tutorial, we'll walk through the process of deploying Ambassador Open Source in Kubernetes for ingress routing. Ambassador OSS provides all the functionality of a traditional ingress controller (i.e., path-based routing) while exposing many additional capabilities such as [authentication](/user-guide/auth-tutorial), URL rewriting, CORS, rate limiting, and automatic metrics collection (the [mappings reference](/reference/mappings) contains a full list of supported options). Note that Ambassador Edge Stack can be used as an [Ingress Controller](/user-guide/ingress-controller).

For more background on Kubernetes ingress, [read this blog post](https://blog.getambassador.io/kubernetes-ingress-nodeport-load-balancers-and-ingress-controllers-6e29f1c44f2d).

Ambassador Open Source is designed to allow service authors to control how their service is published to the Internet. We accomplish this by permitting a wide range of annotations on the *service*, which Ambassador OSS reads to configure its Envoy Proxy. Below, we'll use service annotations to configure Ambassador OSS to map `/httpbin/` to `httpbin.org`.

## 1. Deploying Ambassador Open Source

To deploy Ambassador Open Source in your **default** namespace, first you need to check if Kubernetes has RBAC enabled:

<div class="gatsby-highlight" data-language="shell">
<pre class="language-shell">
<code class="language-shell" id="step1">
kubectl cluster-info dump --namespace kube-system | grep authorization-mode</code></pre></div>
<button onclick="copy_to_clipboard('step1')">Copy to Clipboard</button>
<script>
function copy_to_clipboard(the_id) {
  var copyText = document.getElementById(the_id).innerText;
  const el = document.createElement('textarea');  // Create a <textarea> element
  el.value = copyText;                            // Set its value to the string that you want copied
  el.setAttribute('readonly', '');                // Make it readonly to be tamper-proof
  el.style.position = 'absolute';
  el.style.left = '-9999px';                      // Move outside the screen to make it invisible
  document.body.appendChild(el);                  // Append the <textarea> element to the HTML document
  const selected =
    document.getSelection().rangeCount > 0        // Check if there is any content selected previously
      ? document.getSelection().getRangeAt(0)     // Store selection if found
      : false;                                    // Mark as false to know no selection existed before
  el.select();                                    // Select the <textarea> content
  document.execCommand('copy');                   // Copy - only works as a result of a user action (e.g. click events)
  document.body.removeChild(el);                  // Remove the <textarea> element
  if (selected) {                                 // If a selection existed before copying
    document.getSelection().removeAllRanges();    // Unselect everything on the HTML document
    document.getSelection().addRange(selected);   // Restore the original selection
  }
};
</script>

If you see something like `--authorization-mode=Node,RBAC` in the output, then RBAC is enabled. The majority of current hosted Kubernetes providers (such as GKE) create
clusters with RBAC enabled by default, and unfortunately the above command may not return any information indicating this.

**Note:** If you're using Google Kubernetes Engine with RBAC, you'll need to grant permissions to the account that will be setting up Ambassador OSS. To do this, get your official GKE username, and then grant `cluster-admin` role privileges to that username:

<div class="gatsby-highlight" data-language="shell">
<pre class="language-shell">
<code class="language-shell" id="step2">
kubectl create clusterrolebinding my-cluster-admin-binding --clusterrole=cluster-admin --user=$(gcloud info --format="value(config.account)")</code></pre></div>
<button onclick="copy_to_clipboard('step2')">Copy to Clipboard</button>

If RBAC is enabled:

<div class="gatsby-highlight" data-language="shell">
<pre class="language-shell">
<code class="language-shell" id="step3">
kubectl apply -f https://getambassador.io/yaml/ambassador/ambassador-rbac.yaml</code></pre></div>
<button onclick="copy_to_clipboard('step3')">Copy to Clipboard</button>

Without RBAC, you can use:

<div class="gatsby-highlight" data-language="shell">
<pre class="language-shell">
<code class="language-shell" id="step4">
kubectl apply -f https://getambassador.io/yaml/ambassador/ambassador-no-rbac.yaml</code></pre></div>
<button onclick="copy_to_clipboard('step4')">Copy to Clipboard</button>

We recommend downloading the YAML files and exploring the content. You will see
that an `ambassador-admin` NodePort Service is created (which provides an
Ambassador ODD Diagnostic web UI), along with an ambassador ClusterRole, ServiceAccount and ClusterRoleBinding (if RBAC is enabled). An Ambassador Open Source Deployment is also created.

When not installing Ambassador Open Source into the default namespace you must update the namespace used in the `ClusterRoleBinding` when RBAC is enabled.

For production configurations, we recommend you download these YAML files as your starting point, and customize them accordingly.


## 2. Defining the Ambassador Open Source Service

Ambassador Open Source is deployed as a Kubernetes Service that references the ambassador Deployment you deployed previously. Create the following YAML and put it in a file called`ambassador-service.yaml`.

<div class="gatsby-highlight" data-language="yaml">
<pre class="language-yaml">
<code class="language-yaml" id="step5">
---
apiVersion: v1
kind: Service
metadata:
  name: ambassador
spec:
  type: LoadBalancer
  externalTrafficPolicy: Local
  ports:
   - port: 80
     targetPort: 8080
  selector:
    service: ambassador
</code></pre></div>
<button onclick="copy_to_clipboard('step5')">Copy to Clipboard</button>

Deploy this service with `kubectl`:

<div class="gatsby-highlight" data-language="shell">
<pre class="language-shell">
<code class="language-shell" id="step6">
$ kubectl apply -f ambassador-service.yaml</code></pre></div>
<button onclick="copy_to_clipboard('step6')">Copy to Clipboard</button>

The YAML above creates a Kubernetes service for Ambassador Open Source of type `LoadBalancer`, and configures the `externalTrafficPolicy` to propagate [the original source IP](https://kubernetes.io/docs/tasks/access-application-cluster/create-external-load-balancer/#preserving-the-client-source-ip) of the client. All HTTP traffic will be evaluated against the routing rules you create. Note that if you're not deploying in an environment where `LoadBalancer` is a supported type (such as minikube), you'll need to change this to a different type of service, e.g., `NodePort`.

If you have a static IP provided by your cloud provider you can set as `loadBalancerIP`.

## 3. Creating your first service

Create the following YAML and put it in a file called `tour.yaml`.

<div class="gatsby-highlight" data-language="yaml">
<pre class="language-yaml">
<code class="language-yaml" id="step7">
---
apiVersion: v1
kind: Service
metadata:
  name: tour
spec:
  ports:
  - name: ui
    port: 5000
    targetPort: 5000
  - name: backend
    port: 8080
    targetPort: 8080
  selector:
    app: tour
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tour
spec:
  replicas: 1
  selector:
    matchLabels:
      app: tour
  strategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: tour
    spec:
      containers:
      - name: tour-ui
        image: quay.io/datawire/tour:ui-$tourVersion$
        ports:
        - name: http
          containerPort: 5000
      - name: quote
        image: quay.io/datawire/tour:backend-$tourVersion$
        ports:
        - name: http
          containerPort: 8080
        resources:
          limits:
            cpu: "0.1"
            memory: 100Mi
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: tour-ui
spec:
  prefix: /
  service: tour:5000
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: tour-backend
spec:
  prefix: /backend/
  service: tour:8080
  labels:
    ambassador:
      - request_label:
        - backend
</code></pre></div>
<button onclick="copy_to_clipboard('step7')">Copy to Clipboard</button>

Then, apply it to the Kubernetes with `kubectl`:

<div class="gatsby-highlight" data-language="shell">
<pre class="language-shell">
<code class="language-shell" id="step8">
kubectl apply -f tour.yaml</code></pre></div>
<button onclick="copy_to_clipboard('step8')">Copy to Clipboard</button>

This YAML has also been published so you can deploy it remotely:

<div class="gatsby-highlight" data-language="shell">
<pre class="language-shell">
<code class="language-shell" id="step9">
kubectl apply -f https://getambassador.io/yaml/tour/tour.yaml</code></pre></div>
<button onclick="copy_to_clipboard('step9')">Copy to Clipboard</button>

When the `Mapping` CRDs are applied, Ambassador Open Source will use them to configure routing:

- The first `Mapping` causes traffic from the `/` endpoint to be routed to the `tour-ui` React application.
- The second `Mapping` causes traffic from the `/backend/` endpoint to be routed to the `tour-backend` service.

Note also the port numbers in the `service` field of the `Mapping`. This allows us to use a single service to route to both the containers running on the `tour` pod.

<font color=#f9634E>**Important:**</font>

Routing in Ambassador Open Source can be configured with Ambassador OSS resources as shown above, Kubernetes service annotation, and Kubernetes Ingress resources.

Ambassador OSS ustom resources are the recommended config format and will be used throughout the documentation.

See [configuration format](/reference/config-format) for more information on your configuration options.

## 4. Testing the Mapping

To test things out, we'll need the external IP for Ambassador Open Source (it might take some time for this to be available):

<div class="gatsby-highlight" data-language="shell">
<pre class="language-shell">
<code class="language-shell" id="step10">
kubectl get svc -o wide ambassador</code></pre></div>
<button onclick="copy_to_clipboard('step10')">Copy to Clipboard</button>

Eventually, this should give you something like:

```
NAME         CLUSTER-IP      EXTERNAL-IP     PORT(S)        AGE
ambassador   10.11.12.13     35.36.37.38     80:31656/TCP   1m
```


You should now be able to reach the `tour-ui` application from a web browser:

http://35.36.37.38/

or on minikube:

```shell
$ minikube service list
|-------------|----------------------|-----------------------------|
|  NAMESPACE  |         NAME         |             URL             |
|-------------|----------------------|-----------------------------|
| default     | ambassador-admin     | http://192.168.99.107:30319 |
| default     | ambassador           | http://192.168.99.107:31893 |
|-------------|----------------------|-----------------------------|
```
http://192.168.99.107:31893/

or on Docker for Mac/Windows:

```shell
$ kubectl get svc
NAME               TYPE           CLUSTER-IP       EXTERNAL-IP   PORT(S)          AGE
ambassador         LoadBalancer   10.106.108.64    localhost     80:32324/TCP     13m
ambassador-admin   NodePort       10.107.188.149   <none>        8877:30993/TCP   14m
tour               ClusterIP      10.107.77.153    <none>        80/TCP           13m
kubernetes         ClusterIP      10.96.0.1        <none>        443/TCP          84d
```
http://localhost/

## 5. The Diagnostics Service in Kubernetes

Ambassador Open Source includes an integrated diagnostics service to help with troubleshooting. 

By default, this is exposed to the internet at the URL `http://{{AMBASSADOR_HOST}}/ambassador/v0/diag/`. Go to that URL from a web browser to view the diagnostic UI.

You can change the default so it is not exposed externally by default by setting `diagnostics.enabled: false` in the [ambassador `Module`](/reference/core/ambassador).

<div class="gatsby-highlight" data-language="yaml">
<pre class="language-yaml">
<code class="language-yaml" id="step11">
apiVersion: getambassador.io/v2
kind: Module
metadata:
  name: ambassador
spec:
  config:
    diagnostics:
      enabled: false
</code></pre></div>
<button onclick="copy_to_clipboard('step11')">Copy to Clipboard</button>

After applying this `Module`, to view the diagnostics UI, we'll need to get the name of one of the Ambassador Open Source pods:

```
$ kubectl get pods
NAME                          READY     STATUS    RESTARTS   AGE
ambassador-3655608000-43x86   1/1       Running   0          2m
ambassador-3655608000-w63zf   1/1       Running   0          2m
```

Forwarding local port 8877 to one of the pods:

```
kubectl port-forward ambassador-3655608000-43x86 8877
```

will then let us view the diagnostics at http://localhost:8877/ambassador/v0/diag/.

## 6. Enable HTTPS

The versatile HTTPS configuration of Ambassador Open Source lets it support various HTTPS use cases whether simple or complex.

Follow our [enabling HTTPS guide](/user-guide/tls-termination) to quickly enable HTTPS support for your applications.

## Want more?

For more features, check out the latest build of [Ambassador Edge Stack](/user-guide/install).
