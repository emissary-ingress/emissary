<div style="border: thick solid red">
This method of installation has not been tested and is not supported at this time.
</div>

# Installing Ambassador Edge Stack with Helm

```Note: These instructions do not work with Minikube.```

[Helm](https://helm.sh) is a package manager for Kubernetes. Ambassador Edge Stack is available as a Helm chart if you use Helm for package management. To install with Helm:

1. Install the Ambassador Edge Stack Chart:

   ```
   helm install -n ambassador stable/ambassador
   ```
   
   Details on how to configure the chart, see the [official chart documentation](https://hub.helm.sh/charts/stable/ambassador)

<div style="border: thick solid red">
Next step took service creation from AmbOSS so not sure if this needs to be changed to work for AES
</div>

2. Create your first service(s) based on what you need. For example, here are some of the services you can create with Ambassador Edge Stack:

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
```
<p>
<p>
