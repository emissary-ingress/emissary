# Getting Started with Ambassador Edge Stack

<img src="/doc-images/kubernetes.png"  style="width:60px;height:57px;"/>

Ambassador Edge Stack is designed to run in Kubernetes for production. However, there are a few prerequisites that are important for a successful installation of Ambassador Edge Stack. Make sure you have the following:

* a clean, running [Kubernetes cluster](https://kubernetes.io/docs/setup/), version 1.11 or later
* the command line tool [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)

## 1. Deploying Ambassador Edge Stack to Kubernetes

<div style="border: thick solid red">
Note, the secret.yaml file is temporary during internal Datawire development and can be obtained from the 
<a href="https://drive.google.com/file/d/1q-fmSXU966UtAARrzyCnaKTVbcpkg2n-/view?usp=sharing">Google drive</a>.
</div>

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

### Minikube Users

If you happen to be using Minikube, note that Minikube does not natively support load balancers. Instead, to get the URL of Ambassador from minikube, use the command `minikube service list` and you should see something similar to the folllowing:


(⎈ |minikube:ambassador)$ minikube service list
 |-------------|------------------|--------------------------------|
 |  NAMESPACE  |       NAME       |              URL               |
 |-------------|------------------|--------------------------------|
 | ambassador  | ambassador       | http://192.168.64.2:31230      |
 |             |                  | http://192.168.64.2:31042      |
 | ambassador  | ambassador-admin | No node port                   |
 | ambassador  | ambassador-redis | No node port                   |
 | default     | kubernetes       | No node port                   |
 | kube-system | kube-dns         | No node port                   |
 |-------------|------------------|--------------------------------|
 ```


Use any of the URLs listed next to `ambassador` to access the Ambassador Edge Stack.

## 3. Assign a DNS name (or not)

Navigate to your new IP address in your browser. Assign a DNS name using the providor of your choice to the IP address acquired in Step 2. If you can't/don't want to assign a DNS name, then you can use the IP address you acquired in step 2 instead.

## 4. Complete the install

Go to `http://<your-host-name>` and follow the instructions to complete the install.

You will need to install the `edgectl` tool in order to fully configure the Ambassador Edge Stack UI. If you are having trouble downloading `edgectl`, you can download them directly from [this page](/user-guide/downloads).

## Next Steps


- Join us on [Slack](https://d6e.co/slack);
- Learn how to [add authentication](/user-guide/auth-tutorial) to existing services; or
- Learn how to [add rate limiting](/user-guide/rate-limiting-tutorial) to existing services; or
- Learn how to [add tracing](/user-guide/tracing-tutorial); or
- Learn how to [use gRPC with Ambassador Edge Stack](/user-guide/grpc); or
- Read about [configuring Ambassador Edge Stack](/reference/configuration).
