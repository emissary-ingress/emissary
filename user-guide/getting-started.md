# Deploying Ambassador Edge Stack to Kubernetes

## 1. Deploying Ambassador Edge Stack

<div style="border: thick solid red">
Note, the secret.yaml file is temporary during internal Datawire development and can be obtained from the [Google drive](https://drive.google.com/file/d/1q-fmSXU966UtAARrzyCnaKTVbcpkg2n-/view?usp=sharing).
</div>

```shell
kubectl apply -f secret.yaml
kubectl apply -f https://deploy-preview-91--datawire-ambassador.netlify.com/yaml/aes.yaml
```

## 2. Determine your IP Address

Note that it may take a while for your load balancer ip address to be
provisioned. Repeat this command as necessary until you get an ip
address:

```shell
kubectl get -n ambassador service ambassador -o 'go-template={{range .status.loadBalancer.ingress}}{{print .ip "\n"}}{{end}}'
```

## 3. Assign a DNS name

Assign a DNS name using the providor of your choice to the IP address acquired in Step 2.

## 4. Complete the install

Go to http://<your-host-name> and follow the instructions to complete the install.


## Next Steps

We've just done a quick tour of some of the core features of Ambassador Edge Stack: diagnostics, routing, configuration, and authentication.

- Join us on [Slack](https://d6e.co/slack);
- Learn how to [add authentication](/user-guide/auth-tutorial) to existing services; or
- Learn how to [add rate limiting](/user-guide/rate-limiting-tutorial) to existing services; or
- Learn how to [add tracing](/user-guide/tracing-tutorial); or
- Learn how to [use gRPC with Ambassador Edge Stack](/user-guide/grpc); or
- Read about [configuring Ambassador Edge Stack](/reference/configuration).
