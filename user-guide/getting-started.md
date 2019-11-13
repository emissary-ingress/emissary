# Getting Started with Ambassador Edge Stack

## 1. Deploying Ambassador Edge Stack to Kubernetes

<div style="border: thick solid red">
Note, the secret.yaml file is temporary during internal Datawire development and can be obtained from the 
<a href="https://drive.google.com/file/d/1q-fmSXU966UtAARrzyCnaKTVbcpkg2n-/view?usp=sharing">Google drive</a>.
</div>

## 1. Deploying Ambassador

To deploy Ambassador in your **default** namespace, first you need to check if Kubernetes has RBAC enabled:

```shell
kubectl cluster-info dump --namespace kube-system | grep authorization-mode
```

If you see something like `--authorization-mode=Node,RBAC` in the output, then RBAC is enabled. The majority of current hosted Kubernetes providers (such as GKE) create
clusters with RBAC enabled by default, and unfortunately the above command may not return any information indicating this.

Note: If you're using Google Kubernetes Engine with RBAC, you'll need to grant permissions to the account that will be setting up Ambassador. To do this, get your official GKE username, and then grant `cluster-admin` role privileges to that username:

```
$ kubectl create clusterrolebinding my-cluster-admin-binding --clusterrole=cluster-admin --user=$(gcloud info --format="value(config.account)")
```

If RBAC is enabled:

```shell
kubectl apply -f https://getambassador.io/yaml/ambassador/ambassador-rbac.yaml
```

Without RBAC, you can use:

```shell
kubectl apply -f https://getambassador.io/yaml/ambassador/ambassador-no-rbac.yaml
```

We recommend downloading the YAML files and exploring the content. You will see
that an `ambassador-admin` NodePort Service is created (which provides an
Ambassador Diagnostic web UI), along with an ambassador ClusterRole, ServiceAccount and ClusterRoleBinding (if RBAC is enabled). An Ambassador Deployment is also created.

When not installing Ambassador into the default namespace you must update the namespace used in the `ClusterRoleBinding` when RBAC is enabled.

For production configurations, we recommend you download these YAML files as your starting point, and customize them accordingly.


## 2. Defining the Ambassador Service

Ambassador is deployed as a Kubernetes Service that references the ambassador
Deployment you deployed previously. Create the following YAML and put it in a file called
`ambassador-service.yaml`.

```yaml
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
```

Deploy this service with `kubectl`:

```shell
$ kubectl apply -f ambassador-service.yaml
```

The YAML above creates a Kubernetes service for Ambassador of type `LoadBalancer`, and configures the `externalTrafficPolicy` to propagate [the original source IP](https://kubernetes.io/docs/tasks/access-application-cluster/create-external-load-balancer/#preserving-the-client-source-ip) of the client. All HTTP traffic will be evaluated against the routing rules you create. Note that if you're not deploying in an environment where `LoadBalancer` is a supported type (such as minikube), you'll need to change this to a different type of service, e.g., `NodePort`.

If you have a static IP provided by your cloud provider you can set as `loadBalancerIP`.

## 3. Creating your first service

Create the following YAML and put it in a file called `tour.yaml`.

```yaml
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
apiVersion: getambassador.io/v1
kind: Mapping
metadata:
  name: tour-ui
spec:
  prefix: /
  service: tour:5000
---
apiVersion: getambassador.io/v1
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
```

Then, apply it to the Kubernetes with `kubectl`:

```shell
$ kubectl apply -f tour.yaml
```

This YAML has also been published so you can deploy it remotely:

```
kubectl apply -f https://getambassador.io/yaml/tour/tour.yaml
```

When the `Mapping` CRDs are applied, Ambassador will use them to configure routing:

- The first `Mapping` causes traffic from the `/` endpoint to be routed to the `tour-ui` React application.
- The second `Mapping` causes traffic from the `/backend/` endpoint to be routed to the `tour-backend` service.

Note also the port numbers in the `service` field of the `Mapping`. This allows us to use a single service to route to both the containers running on the `tour` pod.

<font color=#f9634E>**Important:**</font>

Routing in Ambassador can be configured with Ambassador resources as shown above, Kubernetes service annotation, and Kubernetes Ingress resources. 

Ambassador custom resources are the recommended config format and will be used throughout the documentation.

See [configuration format](/reference/config-format) for more information on your configuration options.

## 4. Testing the Mapping

To test things out, we'll need the external IP for Ambassador (it might take some time for this to be available):

```shell
kubectl apply -f secret.yaml && \
kubectl apply -f https://deploy-preview-91--datawire-ambassador.netlify.com/yaml/aes-crds.yaml && \
kubectl wait --for condition=established --timeout=60s crd -lproduct=aes && \
kubectl apply -f https://deploy-preview-91--datawire-ambassador.netlify.com/yaml/aes.yaml && \
kubectl -n ambassador wait --for condition=available --timeout=60s deploy -lproduct=aes
```

## 2. Determine your IP Address

Note that it may take a while for your load balancer IP address to be provisioned. Repeat this command as necessary until you get an IP address:

```shell
kubectl get -n ambassador service ambassador -o 'go-template={{range .status.loadBalancer.ingress}}{{print .ip "\n"}}{{end}}'
```

## 3. Assign a DNS name (or not)

Navigate to your new IP address in your browser. Assign a DNS name using the providor of your choice to the IP address acquired in Step 2. If you can't/don't want to assign a DNS name, then you can use the IP address you acquired in step 2 instead.

## 4. Complete the install

Go to http://&lt;your-host-name&gt; and follow the instructions to complete the install.


## Next Steps

We've just done a quick tour of some of the core features of Ambassador Edge Stack: diagnostics, routing, configuration, and authentication.

- Join us on [Slack](https://d6e.co/slack);
- Learn how to [add authentication](/user-guide/auth-tutorial) to existing services; or
- Learn how to [add rate limiting](/user-guide/rate-limiting-tutorial) to existing services; or
- Learn how to [add tracing](/user-guide/tracing-tutorial); or
- Learn how to [use gRPC with Ambassador Edge Stack](/user-guide/grpc); or
- Read about [configuring Ambassador Edge Stack](/reference/configuration).
