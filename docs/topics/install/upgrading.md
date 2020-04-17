# Upgrading Ambassador Edge Stack

Since Ambassador Edge Stack's configuration is entirely stored in Kubernetes resources, no special process
is necessary to upgrade Ambassador Edge Stack.

The steps to upgrade depend on the method that was used to install Ambassador Edge Stack, as indicated below.

* If you installed using the Operator, then you'll need to [use the Operator to perform the upgrade](aes-operator/#updates-by-the-operator).
To verify whether the Operator was used to install Ambassador Edge Stack, run the following command
to see if it returns resources:
```commandline
$ kubectl get deployment -n ambassador -l 'app.kubernetes.io/name=ambassador,app.kubernetes.io/managed-by in (amb-oper,amb-oper-manifest,amb-oper-helm,amb-oper-azure)' 
NAME               READY   UP-TO-DATE   AVAILABLE   AGE
ambassador         1/1     1            1           ...
```

* If you installed using the Helm chart or `edgectl install`, then you should
[upgrade with the help of Helm](helm/#migrating-to-the-ambassador-edge-stack).
To verify this, run the following command to see if it returns resources:
```commandline
$ kubectl get deployment -n ambassador -l 'app.kubernetes.io/name=ambassador'
NAME               READY   UP-TO-DATE   AVAILABLE   AGE
ambassador         1/1     1            1           ...
```

* Finally, if you installed using manifests, simply run the commands in the following section. To verify whether
manifests were used to install Ambassador Edge Stack, run the following command to see if it returns resources:
```commandline
$ kubectl get deployment -n ambassador -l 'product=aes'
NAME               READY   UP-TO-DATE   AVAILABLE   AGE
ambassador         1/1     1            1           ...
```

If none of the commands above return resources, you probably have an old installation and you should follow
the instructions for [upgrading to Ambassador Edge Stack](upgrade-to-edge-stack/).

## Upgrading an installation with manifests

If you're using the YAML files supplied by Datawire, you'll be able to upgrade simply by repeating
the following `kubectl apply` command:

```shell
kubectl apply -f https://www.getambassador.io/yaml/aes-crds.yaml
```

This will trigger a rolling upgrade of Ambassador Edge Stack.

If you're using your own YAML, check the Datawire YAML to be sure of other changes, but at minimum,
you'll need to change the pulled `image` for the Ambassador Edge Stack container and redeploy.
