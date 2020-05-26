# Edgectl Upgrade: no Ambassador API Gateway installation found

The upgrader has not been able to find an existing API Gateway installation.

## What's next?

* Check that there is an `AmbassadorInstallation` in your cluster:
  ```
  kubectl -n ambassador get ambassadorinstallations.getambassador.io              
  NAME         VERSION   UPDATE-WINDOW   LAST-CHECK             DEPLOYED   DEPLOYED-VERSION   DEPLOYED-FLAVOR
  ambassador   *                         2020-05-21T21:32:24Z   True       1.4.3              OSS
  ```
  The `ambassador` installation should be deployed and with `OSS` flavor.

* Check all the _conditions_ your `AmbassadorInstallation` has passed through with:
  ```commandline
  kubectl get ambassadorinstallations -n ambassador ambassador -o jsonpath='{.status.conditions[?(@.type=="Failed")].message}'
  AuthService(s) exist in the cluster, please remove to upgrade to AES
  ```
  In this example you can see there was a `UpgradePrecondError` error because an `AuthService`
  exist in the cluster.
  
* Perhaps your API Gateway installation is not supported by the upgrader: the upgrade can only
  upgrade installations managed by the Operator.
