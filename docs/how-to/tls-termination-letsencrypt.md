# TLS Termination With Let's Encrypt

With Let's Encrypt, your Ambassador installation can automatically retrieve and renew SSL certificates for free.

The process is as follows:

1. Deploy kube-cert-manager with DNS challenge enabled
2. Create your certificate resource

## 1. Deploy kube-cert-manager

`kube-cert-manager` is responsible for retrieving and renewing SSL certificates from [Let's Encrypt](https://letsencrypt.org/)

[kube-cert-manager](https://github.com/PalmStoneGames/kube-cert-manager) provides the `yaml` files for deploying `kube-cert-manager` and the necessary `Certificate` `CustomResourceDefintiion and the necessary `Certificate` `CustomResourceDefintiion`.

1: Create the Certificate resource type:

```yaml
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: certificates.stable.k8s.psg.io
spec:
  scope: Namespaced
  group: stable.k8s.psg.io
  version: v1
  names:
    kind: Certificate
    plural: certificates
    singular: certificate
```

2: Deploy the `kube-cert-manager` app:

*GKE Users:* Create a Service Account with Read & Write access to Cloud DNS, and then add the JSON key as a secret:

```
kubectl create secret generic kube-cert-manager-key --from-file=./my-kube-cert-manager-account.json
```

*NOTE: Be sure to edit the appropriate fields. This example uses Google Cloud DNS for the challenge provider, but there are [many more options here](https://github.com/PalmStoneGames/kube-cert-manager/blob/master/docs/providers.md)*

*NOTE: This yaml may be out of date. Be sure to check the [deployment-guide](https://github.com/PalmStoneGames/kube-cert-manager/blob/master/docs/deployment-guide.md) for the most up-to-date information*

*NOTE: Be sure to test with the staging server first! Not doing so could result in hitting the Let's Encrypt rate limit, preventing certificates from being issued to your domain for several days*

*NOTE: The image has changed, as PalmStoneGames does not distribute a docker image on the Docker hub any more.*

This will create a 10gb storage volume, provisioned by your cloud provider, and deploy the kube-cert-manager application.

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: kube-cert-manager-certs
  labels:
    app: kube-cert-manager
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  labels:
    app: kube-cert-manager
  name: kube-cert-manager
spec:
  replicas: 1
  template:
    metadata:
      labels:
        app: kube-cert-manager
      name: kube-cert-manager
    spec:
      containers:
        - name: kube-cert-manager
          image: alectroemel/kube-cert-manager:0.5.1
          env:
          - name: "GCE_PROJECT"
            value: "changeme-1234"
          - name: "GOOGLE_APPLICATION_CREDENTIALS"
            value: "/config/my-kube-cert-manager-account.json"
          args:
            - "-data-dir=/var/lib/cert-manager"
            - "-acme-url=https://acme-staging.api.letsencrypt.org/directory"
            # NOTE: the URL above points to the staging server, where you won't get real certs.
            # Uncomment the line below to use the production LetsEncrypt server:
            #- "-acme-url=https://acme-v01.api.letsencrypt.org/directory"
            # You can run multiple instances of kube-cert-manager for the same namespace(s),
            # each watching for a different value for the 'class' label
            #- "-class=default"
            # You can choose to monitor only some namespaces, otherwise all namespaces will be monitored
            #- "-namespaces=default,test"
            # If you set a default email, you can omit the field/annotation from Certificates/Ingresses
            - "-default-email=me@example.com"
            # If you set a default provider, you can omit the field/annotation from Certificates/Ingresses
            - "-default-provider=googlecloud"
          volumeMounts:
            - name: data
              mountPath: /var/lib/cert-manager
        volumes:
        - name: "gce-config"
          secret:
            secretName: kube-cert-manager-key
        - name: "data"
          persistentVolumeClaim:
            claimName: kube-cert-manager-certs
```

3: Create your certificate

This will tell `kube-cert-manager` to create a new Certificate and store it in the secret `ambassador-certs` to create a new Certificate and store it in the secret `ambassador-certs`
```
apiVersion: "stable.k8s.psg.io/v1"
kind: "Certificate"
metadata:
  name: "api-example-com"
  labels:
    stable.k8s.psg.io/kcm.class: "default"
spec:
  domain: "api.example.com"
  email: "you@example.com"
  provider: "googlecloud"
  secretName: ambassador-certs
```

When Ambassador starts, it will notice the `ambassador-certs` secret and turn TLS on.
