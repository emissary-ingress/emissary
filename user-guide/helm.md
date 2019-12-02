# Installing Ambassador Edge Stack with Helm

```Note: These instructions do not work with Minikube.```

[Helm](https://helm.sh) is a package manager for Kubernetes. Ambassador Edge Stack is available as a Helm chart if you use Helm for package management. To install with Helm:

1. Install the Ambassador Edge Stack Chart:

   ```
   helm install -n ambassador stable/ambassador
   ```
   
   Details on how to configure the chart, see the [official chart documentation](https://hub.helm.sh/charts/stable/ambassador)

2. Create your first service(s) based on what you need. 