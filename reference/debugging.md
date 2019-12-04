# Debugging (Advanced)

If the Ambassador Edge Stack is not starting or is not behaving as you would expect, you should visit the Edge Policy Console. This console document covers more advanced use cases and approaches using the command line instead of the Edge Policy Consolee, and assumes that you have either looked at the console or can't access the page due to an issue.

## tl;dr Problem? Start here 

* [Example configuration for debug examples](/reference/debugging#example-config-for-debug-demonstrations)
* Ambassador Edge Stack not starting
  * [Check to see if the Ambassador Edge Stack is running](/reference/debugging#checking-ambassador-edge-stack-is-running) via `kubectl`
  * [Check the logs](/reference/debugging#getting-access-to-the-ambassador-edge-stack-logs)
* The Ambassador Edge Stack is not behaving as expected
  * [Check Ambassador Edge Stack is running correctly](/reference/debugging#checking-ambassador-edge-stack-is-running) via `kubectl`
  * [Check the logs](/reference/debugging#getting-access-to-the-ambassador-edge-stack-logs) (potentially with "Set Debug On" via the Diagnostic Console)
* Ambassdor/Envoy configuration not as unexpected
  * "Set Debug On" (via Diagnostic Console) and [check the (now verbose) logs](/reference/debugging#getting-access-to-the-ambassador-edge-stack-logs)
  * Exec into an Ambassador Edge Stack Pod and [manually verify](/reference/debugging#examining-an-ambassador-edge-stackenvoy-pod-and-container) the generated Envoy configuration
* Mounted TLS certificates not being detected by the Ambassador Edge Stack
  * Exec into an Ambassador Edge Stack Pod and [manually verify](/reference/debugging#examining-an-ambassador-edge-stackenvoy-pod-and-container) that the mount is as expected (and in the correct file system location)
* You want to manually change and experiment with the generated Envoy configuration
  * [Exec into an Ambassador Edge Stack Pod](/reference/debugging#examining-an-ambassador-edge-stackenvoy-pod-and-container) and [manually experiment](/reference/debugging#manually-experimenting-with-ambassador-edge-stack--envoy-configuration) with changing the Envoy configuration and sending a SIGHUP to the parent process

## Example Config for Debug Demonstrations

The following debugging instructions assume that you have deployed the Ambassador Edge Stack and a backend service to a Kubernetes cluster.

e.g. Create a cluster in GKE with RBAC support enabled and your user account configured correctly:

```shell
$ gcloud container clusters create ambassador-demo --preemptible
$ kubectl create clusterrolebinding cluster-admin-binding-new \
--clusterrole cluster-admin --user <your_user_name>
```

Deploy the latest version of the Ambassador Edge Stack:

```shell
$ kubectl apply -f https://getambassador.io/yaml/ambassador/ambassador-rbac.yaml
```
Next, create an Ambassador Edge Stack Service and deploy a basic `httpbin` Ambassador Edge Stack Mapping
e.g. save this YAML to a file named ```ambassador-services.yaml```

```yaml
---
apiVersion: v1
kind: Service
metadata:
  name: ambassador
spec:
  type: LoadBalancer
  ports:
   - port: 80
  selector:
    service: ambassador
---
apiVersion: getambassador.io/v2
kind:  Mapping
metadata:
  name:  httpbin
spec:
  prefix: /httpbin/
  service: httpbin.org
  host_rewrite: httpbin.org
```
And apply this into your cluster, e.g.:

```shell
$ kubectl apply -f ambassador-services.yaml
```

### Checking Ambassador Edge Stack is running

If you cannot access the [diagnostics console](/reference/diagnostics) via ```kubectl port-forward <ambassador_pod_name> 8877```
the first thing to check is that the Ambassador Edge Stack is running. This can be achieved via
the standard Kubernetes commands.

First, check the Deployment

```shell
$ kubectl get deployments
NAME         DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
ambassador   3         3         3            3           1m
```

If after a brief period of time to allow for the Ambassador Edge Stack to initialize the "desired" number of replicas does not equal the "current" or "available" number, then you will also want to check the associated Pods:

```shell
$ kubectl get pods
NAME                         READY     STATUS    RESTARTS   AGE
ambassador-85c4cf67b-4pfj2   1/1       Running   0          1m
ambassador-85c4cf67b-fqp9g   1/1       Running   0          1m
ambassador-85c4cf67b-vg6p5   1/1       Running   0          1m
```

If any of the Pods have not started you can "Describe" both the Deployment and individual Pods.

When describing the Deployment, pay particular attention to the "Replicas" (close to the topi of the output) and the "Events" log (close to the bottom of the output). *The "Events" log will often show information like a failed image pull, RBAC issues, or a lack of cluster resources.*

```shell
$ kubectl describe deployment ambassador
Name:                   ambassador
Namespace:              default
CreationTimestamp:      Mon, 15 Oct 2018 13:26:40 +0100
Labels:                 service=ambassador
Annotations:            deployment.kubernetes.io/revision=1
                       kubectl.kubernetes.io/last-applied-configuration={"apiVersion":"apps/v1","kind":"Deployment","metadata":{"annotations":{},"name":"ambassador","namespace":"default"},"spec":{"replicas":3,"te...
Selector:               service=ambassador
Replicas:               3 desired | 3 updated | 3 total | 3 available | 0 unavailable
StrategyType:           RollingUpdate

...

Pod Template:
 Labels:           service=ambassador
 Annotations:      sidecar.istio.io/inject=false
 Service Account:  ambassador
 Containers:
  ambassador:
   Image:       quay.io/datawire/ambassador:0.40.0
   Ports:       8080/TCP, 8443/TCP, 8877/TCP
   Host Ports:  0/TCP, 0/TCP, 0/TCP
   Limits:
     cpu:     1
     memory:  400Mi
   Requests:
     cpu:      200m
     memory:   100Mi
   Liveness:   http-get http://:8877/ambassador/v0/check_alive delay=30s timeout=1s period=3s #success=1 #failure=3
   Readiness:  http-get http://:8877/ambassador/v0/check_ready delay=30s timeout=1s period=3s #success=1 #failure=3
   Environment:
     AMBASSADOR_NAMESPACE:   (v1:metadata.namespace)
   Mounts:                  <none>
 Volumes:                   <none>

...

Conditions:
  Type           Status  Reason
  ----           ------  ------
  Available      True    MinimumReplicasAvailable
OldReplicaSets:  <none>
NewReplicaSet:   ambassador-85c4cf67b (3/3 replicas created)
Events:
  Type    Reason             Age   From                   Message
  ----    ------             ----  ----                   -------
  Normal  ScalingReplicaSet  2m    deployment-controller  Scaled up replica set ambassador-85c4cf67b to 3
```

You can also describe individual Pods, paying particular attention to the "Status" field (at the top of the output) and the "Events" log (at the bottom of the output). *The "Events" log will often show issues such as image pull failures, volume mount issues, and container crash loops,* e.g.:

```shell
$ kubectl get pods
NAME                         READY     STATUS    RESTARTS   AGE
ambassador-85c4cf67b-4pfj2   1/1       Running   0          3m


$ kubectl describe pods ambassador-85c4cf67b-4pfj2
Name:           ambassador-85c4cf67b-4pfj2
Namespace:      default
Node:           gke-ambassador-demo-default-pool-912378e5-dkxc/10.128.0.2
Start Time:     Mon, 15 Oct 2018 13:26:40 +0100
Labels:         pod-template-hash=417079236
                service=ambassador
Annotations:    sidecar.istio.io/inject=false
Status:         Running
IP:             10.60.0.5
Controlled By:  ReplicaSet/ambassador-85c4cf67b
Containers:
  ambassador:
    Container ID:   docker://33ab16fe9f02bb425dd03a501b70c67eb41fd5831ff68e064f64965584e7cd43
    Image:          quay.io/datawire/ambassador:0.40.0

...

Events:
  Type    Reason                 Age   From                                                     Message
  ----    ------                 ----  ----                                                     -------
  Normal  Scheduled              4m    default-scheduler                                        Successfully assigned ambassador-85c4cf67b-4pfj2 to gke-ambassador-demo-default-pool-912378e5-dkxc
  Normal  SuccessfulMountVolume  4m    kubelet, gke-ambassador-demo-default-pool-912378e5-dkxc  MountVolume.SetUp succeeded for volume "ambassador-token-tmk94"
  Normal  Pulling                4m    kubelet, gke-ambassador-demo-default-pool-912378e5-dkxc  pulling image "quay.io/datawire/ambassador:0.40.0"
  Normal  Pulled                 4m    kubelet, gke-ambassador-demo-default-pool-912378e5-dkxc  Successfully pulled image "quay.io/datawire/ambassador:0.40.0"
  Normal  Created                4m    kubelet, gke-ambassador-demo-default-pool-912378e5-dkxc  Created container
  Normal  Started                4m    kubelet, gke-ambassador-demo-default-pool-912378e5-dkxc  Started container
```

### Getting Access to the Ambassador Edge Stack Logs

The Ambassador Edge Stack logs can provide a lot of information if something isn't behaving as expected. There can be a lot of text to parse (especially when running in debug mode), but key information to look out for is the Ambassador Edge Stack process restarting unexpectedly, or malformed Envoy configuration.

In order to view the logs you will need to target an individual Ambassador Edge Stack Pod. e.g.:

```shell
$ kubectl get pods
NAME                         READY     STATUS    RESTARTS   AGE
ambassador-85c4cf67b-4pfj2   1/1       Running   0          3m
$
$ kubectl logs ambassador-85c4cf67b-4pfj2
2018-10-10 12:26:50 kubewatch 0.40.0 INFO: generating config with gencount 1 (0 changes)
/usr/lib/python3.6/site-packages/pkg_resources/__init__.py:1235: UserWarning: /ambassador is writable by group/others and vulnerable to attack when used with get_resource_filename. Consider a more secure location (set with .set_extraction_path or the PYTHON_EGG_CACHE environment variable).
  warnings.warn(msg, UserWarning)
2018-10-10 12:26:51 kubewatch 0.40.0 INFO: Scout reports {"latest_version": "0.40.0", "application": "ambassador", "notices": [], "cached": false, "timestamp": 1539606411.061929}

2018-10-10 12:26:54 diagd 0.40.0 [P15TMainThread] INFO: thread count 3, listening on 0.0.0.0:8877
[2018-10-10 12:26:54 +0000] [15] [INFO] Starting gunicorn 19.8.1
[2018-10-10 12:26:54 +0000] [15] [INFO] Listening at: http://0.0.0.0:8877 (15)
[2018-10-10 12:26:54 +0000] [15] [INFO] Using worker: threads
[2018-10-10 12:26:54 +0000] [42] [INFO] Booting worker with pid: 42
2018-10-10 12:26:54 diagd 0.40.0 [P42TMainThread] INFO: Starting periodic updates
[2018-10-10 12:27:01.977][21][info][main] source/server/drain_manager_impl.cc:63] shutting down parent after drain
```

By using the [Ambassador diagnostics console](/reference/diagnostics) you can click a button to "Set Debug On", and this causes Ambassador Edge Stack to generate a lot more logging. This can be useful when tracking down a particularly subtle bug.

### Examining an Ambassador Edge Stack/Envoy Pod and Container

It can sometimes be useful to examine the contents of the Ambassador Edge Stack Pod, for example, to check volume mounts are correct (e.g. TLS certificates are present in the required directory), to determine the latest Ambassador Edge Stack configuration has been sent to the Pod, or that the generated Envoy configuration is correct (or as expected).

You can look into an Ambassador Edge Stack Pod by using ```kube-exec``` and the ```/bin/sh``` shell contained within the Ambassador Edge Stack container. e.g.:

```shell
$ kubectl get pods
NAME                         READY     STATUS    RESTARTS   AGE
ambassador-85c4cf67b-4pfj2   1/1       Running   0          14m
ambassador-85c4cf67b-fqp9g   1/1       Running   0          14m
ambassador-85c4cf67b-vg6p5   1/1       Running   0          14m
$
$ kubectl exec -it ambassador-85c4cf67b-4pfj2 -- /bin/sh
/ambassador # pwd
/ambassador
/ambassador # ls -lsa
total 84
     4 drwxrwxr-x    1 root     root          4096 Oct 15 12:35 .
     4 drwxr-xr-x    1 root     root          4096 Oct 15 12:26 ..
     4 drwxr-xr-x    4 root     root          4096 Oct 15 12:26 ambassador-0.40.0-py3.6.egg-tmp
     4 drwxrwxr-x    1 root     root          4096 Sep 25 20:29 ambassador-config
     4 drwxr-xr-x    2 root     root          4096 Oct 15 12:26 ambassador-config-1
     4 drwxr-xr-x    2 root     root          4096 Oct 15 12:35 ambassador-config-2
     4 drwxrwxr-x    1 root     root          4096 Sep 25 20:29 ambassador-demo-config
     8 -rwxr-xr-x    1 root     root          4179 Sep 25 20:28 entrypoint.sh
     4 -rw-r--r--    1 root     root          3322 Oct 15 12:26 envoy-1.json
     8 -rw-r--r--    1 root     root          4101 Oct 15 12:35 envoy-2.json
     8 -rw-rw-r--    1 root     root          5245 Sep 25 20:28 hot-restarter.py
    20 -rw-rw-r--    1 root     root         16952 Sep 25 20:28 kubewatch.py
     4 -rwxrwxr--    1 root     root           175 Sep 25 20:28 requirements.txt
     4 -rwxr-xr-x    1 root     root           997 Sep 25 20:28 start-envoy.sh
```
The above output shows a typical file list from a pre-0.50 Ambassador Edge Stack instance. The `ambassador -config-X` directories contain the Ambassador Edge Stack configuration that was specified during each update of Ambassador Edge Stack via Kubernetes config files, with the higher number indicating the more recent configuration (as verified by the directory timestamps). The easy method to determine the latest configuration is to look for the `ambassador-config-X` directory with the highest number.

```shell
/ambassador # ls ambassador-config-2
Httpbin-default.yaml

/ambassador # cat ambassador-config-2/Httpbin-default.yaml

---
apiVersion: v0.1
kind: Pragma
ambassador_id: default
source: "service httpbin, namespace default"
autogenerated: true
---
apiVersion: getambassador.io/v2
kind:  Mapping
name:  httpbin_mapping
prefix: /httpbin/
service: httpbin.org:80
host_rewrite: httpbin.org
```


The Envoy Proxy configuration that was generated from the Ambassador Edge Stack configuration is found in corresponding `envoy-X.json` file (where the number matches the `ambassador-config-X` directory number). The contents of the Envoy configuration files can be very useful when looking for subtle mapping issues or bugs.

```shell
/ambassador # cat envoy-2.json

{
  "listeners": [

    {
      "address": "tcp://0.0.0.0:8080",

      "filters": [
        {
          "type": "read",
          "name": "http_connection_manager",
          "config": {"codec_type": "auto",
            "stat_prefix": "ingress_http",
            "access_log": [
              {
```

### Manually Experimenting with Ambassador Edge Stack / Envoy configuration

If the generated Envoy configuration is not looking as expected, you can manually tweak this and restart the Envoy process. The general approach to this is to scale down the Ambassador Edge Stack Deployment to a single Pod in order to send all Ambassador Edge Stack traffic through this single instance (which is not recommended in production!), exec into the Pod and make the modification to the `envoy/envoy.json`. Then, restart the `ambex` process which will pass the updated `envoy.json` to Envoy. You can do this by getting the PID of `ambex` with `ps -ef | grep ambex`. Then run `kill -HUP $AMBEX_PID` to restart `ambex`.

```shell
$ kubectl scale deployment ambassador --replicas=1
deployment.apps/ambassador scaled
 tmp $ kubectl get pods
NAME                         READY     STATUS        RESTARTS   AGE
ambassador-85c4cf67b-4pfj2   1/1       Running       0          30m
ambassador-85c4cf67b-fqp9g   1/1       Terminating   0          30m
ambassador-85c4cf67b-vg6p5   1/1       Terminating   0          30m
```

Wait for the scale down to complete, and then modify the Envoy config e.g.,:

```shell
$ kubectl exec -it ambassador-85c4cf67b-4pfj2 -- /bin/sh
/ambassador $ ls -lsa
total 64
     8 drwxrwxr-x    1 root     root          4096 Apr  5 21:01 .
     4 drwxr-xr-x    1 root     root          4096 Apr  5 21:01 ..
     4 drwxr-xr-x    3 8888     root          4096 Apr  5 21:01 .cache
     4 drwxrwxr-x    1 root     root          4096 Apr  5 17:38 ambassador-config
     4 drwxrwxr-x    1 root     root          4096 Apr  5 17:38 ambassador-demo-config
     4 -rw-r--r--    1 8888     root             2 Apr  5 21:01 ambex.pid
     4 -rw-r--r--    1 8888     root          1626 Apr  5 21:01 bootstrap-ads.json
     4 drwxrwxr-x    1 root     root          4096 Apr  5 17:38 demo-services
     8 -rwxr-xr-x    1 root     root          6589 Apr  5 17:37 entrypoint.sh
     4 drwxrwxr-x    1 root     root          4096 Apr  5 21:01 envoy
     4 -rwxr-xr-x    1 root     root           584 Apr  5 17:37 kick_ads.sh
     4 -rwxr-xr-x    1 root     root          4007 Apr  5 17:37 kubewatch.py
     4 -rwxr-xr-x    1 root     root           484 Apr  5 17:37 post_update.py
     4 drwxr-xr-x    3 8888     root          4096 Apr  5 21:01 snapshots
/ambassador $ vi envoy/envoy.json
/ambassador $ ps -ef | grep ambex
   33 8888     21:56 ambex /ambassador/envoy
  122 8888      0:00 grep ambex
/ambassador $ kill -HUP 33
```

Be aware that even though you have modified the configuration files, the Ambassador Edge Stack Diagnostic Console may not accurately reflect your updates. In order to determine that the restart was successful with the correct configuration, you can ensure that the "Set Debug On" has been enabled via the Diagnostic Console and you can follow the Ambassador Edge Stack/Envoy logs to see the new configuration has been loaded.
