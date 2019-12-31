# Debugging

If you’re experiencing issues with Ambassador and cannot diagnose the issue through the "Diagnostics" tab from the [Edge Policy Console](../../about/edge-policy-console), this document covers various approaches and advanced use cases for debugging Ambassador issues.

First, create an example configuration for debugging demonstrations that are available throughout this document.

Then, complete the following debugging options:

* Check Ambassador Status
* Review Ambassador Logs
* Examine Pod and Container Contents
* Customize the Envoy Configuration

## Example Configuration

To see a demonstration of how you can debug your Ambassador instance, follow the instructions to create an example mapping configuration.

Note: The following assumes that you deployed Ambassador and the following services from the [quick start installation guide](../../user-guide/install) to a Kubernetes cluster.

**In the command line:**

1. Create a cluster in GKE with RBAC support enabled, and your user account configured correctly. Then run the following:

    ```shell
    $ gcloud container clusters create ambassador-demo --preemptible
    $ kubectl create clusterrolebinding cluster-admin-binding-new \
    --clusterrole cluster-admin --user <your_user_name>
    ```

2. Deploy the latest version of Ambassador:

    ```shell
    $ kubectl apply -f https://getambassador.io/yaml/ambassador/ambassador-rbac.yaml
    ```

3. Next, create an Ambassador Service and deploy a basic `httpbin` Ambassador Mapping by saving the following YAML to a file named `ambassador-services.yaml`

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
    apiVersion: getambassador.io/v1
    kind:  Mapping
    metadata:
    name:  httpbin
    spec:
    prefix: /httpbin/
    service: httpbin.org
    host_rewrite: httpbin.org
    ```

4. Apply this into your cluster with the following command:

    ```shell
    $ kubectl apply -f ambassador-services.yaml
    ```

You should now be able to utilize this example mapping for debugging demonstrations and purposes.

## Check Ambassador Status

First, check to see if the [Diagnostics](/reference/diagnostics) service is reachable with the command: `kubectl port-forward <ambassador_pod_name> 8877`

If it is successful, try to diagnose your original issue with the Diagnostics Console.

**If it is not successful, complete the following to see if Ambassador is running:**

1. Check the Ambassador deployment with the following: `kubectl get deployments`
2. After a brief period, the terminal will print something similar to the following:

    ```shell
    kubectl get deployments
    NAME         DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
    ambassador   3         3         3            3           1m
    ```

3. Check that the “desired” number of Pods equals the “current” or “available” number of pods. If they are **not** equal, check the status of the associated pods with the following command: `kubectl get pods`

    The terminal should print something similar to the following:

    ```shell
    $ kubectl get pods
    NAME                         READY     STATUS    RESTARTS   AGE
    ambassador-85c4cf67b-4pfj2   1/1       Running   0          1m
    ambassador-85c4cf67b-fqp9g   1/1       Running   0          1m
    ambassador-85c4cf67b-vg6p5   1/1       Running   0          1m
    ```

4. If any of the Pods have a status of “not started,” use the following command to “describe” the Deployment pods: `kubectl describe deployment ambassador`

    * Look for data in the “Replicas” field near the top of the output. For example: 
        `Replicas: 3 desired | 3 updated | 3 total | 3 available | 0 unavailable`

    * Look for data in the “Events” log field near the bottom of the output, which often displays data such as a fail image pull, RBAC issues, or a lack of cluster resources. For example:
        ```shell
        Events:
        Type    Reason         Age   From                   Message
        ----    ------         ----  ----                   -------
        Normal  ScalingReplicaSet  2m    deployment-controller  Scaled up replica set ambassador-85c4cf67b to 3
        ```

5. Additionally, use the following command to “describe” the individual pods: `kubectl get pods` and then `kubectl describe pods ambassador-<name>`

    * Look for data in the “Status” field near the top of the output. For example, `Status: Running`

    * Look for data in the “Events” field near the bottom of the output, as it will often show issues such as image pull failures, volume mount issues, and container crash loops. For example:
        ```
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

In both the Deployment pod and the individual pods, take the necessary action to address any discovered issues.

## Review Ambassador Logs

The Ambassador logging can provide information on anything that might be abnormal or malfunctioning. While there may be a large amount of data to sort through, look for key errors such as the Ambassador process restarting unexpectedly, or a malformed Envoy configuration.

You can turn on Debug mode in the [Edge Policy Console](/about/edge-policy-console), which generates verbose logging data that can be useful when trying to find a subtle error or bug.

1. Use the following command to target an individual Ambassador Pod: `kubectl get pods`

    The terminal will print something similar to the following:

    ```shell
    $ kubectl get pods
    NAME                         READY     STATUS    RESTARTS   AGE
    ambassador-85c4cf67b-4pfj2   1/1       Running   0          3m
    ```

2. Then, run the following: `kubectl logs ambassador-<pod>`

The terminal will print something similar to the following:

    ```
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

## Examine Pod and Container Contents

You can examine the contents of the Ambassador Pod for issues, such as if volume mounts are correct and TLS certificates are present in the required directory, to determine if the Pod has the latest Ambassador configuration, or if the generated Envoy configuration is correct or as expected. In these instructions, we will look for problems related to the Envoy configuration.

1. To look into an Ambassador Pod, use the container shell with the `kube-exec` and the `/bin/sh` commands. For example, `kubectl exec -it ambassador-<pod> -- /bin/sh`

    The terminal will print a typical file list from a pre-0.50 Ambassador instance, similar to the following:

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

The `ambassador-config-N` lines indicate the directories which contain the specific Ambassador configuration used in YAML files during updates. Higher numbers indicate the most recent configuration, as verified by the directory timestamp.

2. Determine the latest configuration. An easy method is to simply look for the directory with the highest number. For example, `ambassador-config-2` is the most recent Ambassador configuration.
3. Print the contents of the latest configuration: `/ambassador # ls ambassador-config-2`
4. Navigate into the YAML file within that directory: `/ambassador # cat ambassador-config-2/Httpbin-default.yaml`

    The terminal will print something similar to the following:

    ```
    ---
    apiVersion: v0.1
    kind: Pragma
    ambassador_id: default
    source: "service httpbin, namespace default"
    autogenerated: true
    ---
    apiVersion: ambassador/v1
    kind:  Mapping
    name:  httpbin_mapping
    prefix: /httpbin/
    service: httpbin.org:80
    host_rewrite: httpbin.org
    ```

5. Fix any errors you find in this YAML file.
6. Next, find the corresponding Envoy file. The file will be titled `envoy-N.json` where N matches the number of the `ambassador-config-N` directory number.
7. Print the contents of the corresponding Envoy file that is generated during Ambassador configuration with the following command: `/ambassador # cat envoy-2.json`

    The terminal will print something similar to the following:

    ```json
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

The contents of the Envoy configuration files can be very useful when looking for subtle mapping issues or bugs.

## Customize the Envoy Configuration

If the generated Envoy configuration does not look correct, you can edit and restart the Envoy process. We do not recommend doing this in production.

1. Scale down the Ambassador Deployment to a single pod in order to send all Ambassador traffic through a single instance. Use the following command: `kubectl scale deployment ambassador --replicas=1`
2. Wait for the scale down to complete, and then use the exec command to access the Pod: `kubectl exec -it ambassador-85c4cf67b-4pfj2 -- /bin/sh`
3. Modify the `envoy/envoy.json` file: `/ambassador $ vi envoy/envoy.json`
4. Get the PID of the `ambex` process with: `ps -ef | grep ambex`
5. Then, restart the `ambex` process which will pass the updated `envoy.json` to Envoy with: `kill -HUP $AMBEX_PID`
6. In your Edge Policy Console, go to **Diagnostics > Logging** and choose the “set log level to debug” option.
7. Verify that the restart and new configuration was successful with the correct configuration by following the Ambassador/Envoy logs.

Be aware that even though you have modified the configuration files, the Edge Policy Console and/or the Diagnostics service may not accurately reflect your updates.

The command history will look similar to the following:

```shell
$ kubectl scale deployment ambassador --replicas=1
deployment.apps/ambassador scaled
tmp $ kubectl get pods
NAME                         READY     STATUS        RESTARTS   AGE
ambassador-85c4cf67b-4pfj2   1/1       Running       0          30m
ambassador-85c4cf67b-fqp9g   1/1       Terminating   0          30m
ambassador-85c4cf67b-vg6p5   1/1       Terminating   0          30m

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
