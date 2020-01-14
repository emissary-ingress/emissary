# Debugging

If you’re experiencing issues with the Ambassador Edge Stack and cannot diagnose the issue through the "Diagnostics" tab from the [Edge Policy Console](../../about/edge-policy-console), this document covers various approaches and advanced use cases for debugging Ambassador issues.

We assume that you already have a running Ambassador installation in the following sections.

## Check Ambassador Status

First, check to see if the [Diagnostics](../diagnostics) service is reachable. If it is successful, try to diagnose your original issue with the Diagnostics Console.

**If it is not successful, complete the following to see if Ambassador is running:**

1. Get a list of Pods in the `ambassador` namespace with `kubectl get pods -n ambassador`.

    The terminal should print something similar to the following:

    ```console
    $ kubectl get pods -n ambassador
    NAME                         READY     STATUS    RESTARTS   AGE
    ambassador-85c4cf67b-4pfj2   1/1       Running   0          1m
    ambassador-85c4cf67b-fqp9g   1/1       Running   0          1m
    ambassador-85c4cf67b-vg6p5   1/1       Running   0          1m
    ```

2. Then, check the Ambassador Deployment with the following: `kubectl get -n ambassador deployments`

    After a brief period, the terminal will print something similar to the following:

    ```console
    $ kubectl get -n ambassador deployments
    NAME         DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
    ambassador   3         3         3            3           1m
    ```

3. Check that the “desired” number of Pods equals the “current” and “available” number of Pods. If they are **not** equal, check the status of the associated Pods with the following command: `kubectl get pods -n ambassador`.
4. Use the following command for details about the history of the Deployment: `kubectl describe -n ambassador deployment ambassador`

    * Look for data in the “Replicas” field near the top of the output. For example: 
        `Replicas: 3 desired | 3 updated | 3 total | 3 available | 0 unavailable`

    * Look for data in the “Events” log field near the bottom of the output, which often displays data such as a fail image pull, RBAC issues, or a lack of cluster resources. For example:

        ```
        Events:
        Type    Reason              Age     From                      Message
        ----    ------              ----    ----                      -------
        Normal  ScalingReplicaSet    2m     deployment-controller      Scaled up replica set ambassador-85c4cf67b to 3
        ```

5. Additionally, use the following command to “describe” the individual Pods: `kubectl describe pods -n ambassador <ambassador-pod-name>`

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

In both the Deployment Pod and the individual Pods, take the necessary action to address any discovered issues.

## Review Ambassador Logs

The Ambassador logging can provide information on anything that might be abnormal or malfunctioning. While there may be a large amount of data to sort through, look for key errors such as the Ambassador process restarting unexpectedly, or a malformed Envoy configuration.

You can turn on Debug mode in the [Edge Policy Console](../../about/edge-policy-console), which generates verbose logging data that can be useful when trying to find a subtle error or bug.

1. Use the following command to target an individual Ambassador Pod: `kubectl get pods -n ambassador`

    The terminal will print something similar to the following:

    ```console
    $ kubectl get pods -n ambassador
    NAME                         READY     STATUS    RESTARTS   AGE
    ambassador-85c4cf67b-4pfj2   1/1       Running   0          3m
    ```

2. Then, run the following: `kubectl logs -n ambassador <ambassador-pod-name>`

The terminal will print something similar to the following:

    ```console
    $ kubectl logs -n ambassador ambassador-85c4cf67b-4pfj2
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

1. To look into an Ambassador Pod, use the container shell with the `kube-exec` and the `/bin/sh` commands. For example, `kubectl exec -it -n ambassador <ambassador-pod-name> -- /bin/sh`
2. Determine the latest configuration. If you haven't overridden the configuration directory, the latest configuration will be in `/ambassador/snapshots`. If you have overridden it, Ambassador saves  configurations in `$AMBASSADOR_CONFIG_BASE_DIR/snapshots`.

    In the snapshots directory:

    * `snapshot.yaml` contains the full input configuration that Ambassador has found;
    * `aconf.json` contains the Ambassador configuration extracted from the snapshot;
    * `ir.json` contains the IR constructed from the Ambassador configuration; and
    * `econf.json`contains the Envoy configuration generated from the IR.

    The Envoy configuration is then split into `$AMBASSADOR_CONFIG_BASE_DIR/bootstrap-ads.json` and `$AMBASSADOR_CONFIG_BASE_DIR/envoy/envoy.json`, which are the actual input files handed to Envoy.

    In the snapshots directory, the current configuration will be stored in files with no digit suffix, and older configurations have increasing numbers. For example, `ir.json` is current, `ir-1.json` is the next oldest, then `ir-2.json`, etc.

5. If something is wrong with `snapshot` or `aconf`, there is an issue with your configuration. If something is wrong with `ir` or `econf`, you should [open an issue on Github](https://github.com/datawire/ambassador/issues/new/choose).
6. To find the main configuration for Envoy, run: `$AMBASSADOR_CONFIG_BASE_DIR/envoy/envoy.json`.
7. For the bootstrap configuration, which has details about Envoy statistics, logging, and auth, run: `$AMBASSADOR_CONFIG_BASE_DIR/bootstrap-ads.json`.
8. For further details, you can print the Envoy configuration that is geenerated during the Ambassador configuration. The file will be titled `envoy-N.json` where N matches the number of the `ambassador-config-N` directory number. Run the following command: `# cat envoy-2.json`

    The terminal will print something similar to the following:

    ```console
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
