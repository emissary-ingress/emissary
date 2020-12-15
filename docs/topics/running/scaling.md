# Scaling Ambassador

Scaling any cloud native application is inherently domain specific, however the content here
reflects common issues, tips, and tricks that come up frequently.

## Performance Dimensions

The performance of Ambassador Edge Stack's control plane can be characterized along a number of
different dimensions:

 - The number of `TLSContext` resources.
 - The number of `Host` resources.
 - The number of `Mapping` resources per `Host` resource.
 - The number of unconstrained `Mapping` resources (these will apply to all `Host` resources).

If your application involves a larger than average number of any of the above resources, you may
find yourself in need of some of the content in this section.

## Mysterious Pod Restarts (aka Pushing the Edge of the Envelope)

Whether your application is growing organically or whether you are deliberately scale testing, it's
helpful to recognize how Ambassador Edge Stack behaves as it reaches the edge of its performance
envelope along any of these dimensions.

As Ambassador Edge Stack approaches the edge if its performance envelope, it will often manifest as
mysterious pod restarts triggered by Kubernetes. This does not always mean there is a problem, it
could just mean you need to tune some of the resource limits set in your deployment. When it comes
to scaling, Kubernetes will generally kill an Ambassador pod for one of two reasons: exceeding
memory limits or failed liveness/readiness probes. See the [Memory Limits](#memory-limits),
[Liveness Probes](#liveness-probes), and [Readiness Probes](#readiness-probes)
sections for more on how to cope with these situations.

## Memory Limits

Ambassador Edge Stack can grow in memory usage and be killed by Kubernetes if it exceeds the limits
defined in its pod spec. When this happens it is confusing and difficult to catch because the only
indication that this has occurred is the pod transitioning momentarily into the `OOMKilled`
state. The only way to actually observe this is if you are lucky enough to be running the following
command (or have similar monitoring configured) when Ambassador gets `OOMKilled`:

```bash
    kubectl get pods -n ambassador -w
```

In order to take the luck out of the equation, Ambassador Edge Stack will periodically log its
memory usage so you can see in the logs if memory limits might be a problem and require adjustment:

```
2020/11/26 22:35:20 Memory Usage 0.56Gi (28%)
    PID 1, 0.22Gi: busyambassador entrypoint 
    PID 14, 0.04Gi: /usr/bin/python /usr/bin/diagd /ambassador/snapshots /ambassador/bootstrap-ads.json /ambassador/envoy/envoy.json --notices /ambassador/notices.json --port 8004 --kick kill -HUP 1 
    PID 16, 0.12Gi: /ambassador/sidecars/amb-sidecar 
    PID 37, 0.07Gi: /usr/bin/python /usr/bin/diagd /ambassador/snapshots /ambassador/bootstrap-ads.json /ambassador/envoy/envoy.json --notices /ambassador/notices.json --port 8004 --kick kill -HUP 1 
    PID 48, 0.08Gi: envoy -c /ambassador/bootstrap-ads.json --base-id 0 --drain-time-s 600 -l error 
```

In general you should try to keep Ambassador's memory usage below 50% of the pod's limit. This may
seem like a generous safety margin, but when reconfiguration occurs, Ambassador requires additional
memory to avoid disrupting active connections. At each reconfiguration, Ambassador keeps around the
old version for the duration of the configured drain time. See
[AMBASSADOR_DRAIN_TIME](#ambassador_drain_time) for more details on how to tune this
behavior.

Ambassador Edge Stack's exact memory usage depends on (among other things) how many `Host` and
`Mapping` resources are defined in your cluster. If this number has grown over time, you may need to
increase the memory limit defined in your deployment.

## Liveness Probes

Ambassador defines the `/ambassador/v0/check_alive` endpoint on port `8877` for use with Kubernetes
liveness probes. See the Kubernetes documentation for more details on [HTTP liveness probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-a-liveness-http-request).

Kubernetes will restart the Ambassador pod if it fails to get a 200 result from the endpoint. If
this happens it won't necessarily show up in an easily recognizable way in the pod logs. You can
look for Kubernetes events to see if this is happening. Use `kubectl describe pod -n ambassador` or
`kubectl get events -n ambassador` or equivalent.

The purpose of liveness probes is to rescue an Ambassador instance that is wedged, however if
liveness probes are too sensitive they can take out Ambassador instances that are functioning
normally. This is more prone to happen as the number of Ambassador inputs increase. The
`timeoutSeconds` and `failureThreshold` fields of the Ambassador deployment's liveness Probe
determines how tolerant Kubernetes is with its probes. If you observe pod restarts along with
`Unhealthy` events, try tuning these fields upwards from their default values. See the Kubernetes documentation for more details on [tuning probes](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.19/#probe-v1-core).

Note that whatever changes you make to Ambassador's liveness probes should most likely be made to
its readiness probes also.

## Readiness Probes

Ambassador defines the `/ambassador/v0/check_ready` endpoint on port `8877` for use with Kubernetes
readiness probes. See the Kubernetes documentation for more details on [readiness probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-readiness-probes).

Kubernetes uses readiness checks to prevent traffic from going to pods that are not ready to handle
requests. The only time Ambassador cannot usefully handle requests is during initial startup when it
has not yet loaded all the routing information from Kubernetes and/or consul. During this bootstrap
period there is no guarantee Ambassador would know where to send a given request. The `check_ready`
endpoint will only return 200 when all routing information has been loaded. After the initial
bootstrap period it behaves identically to the `check_alive` endpoint.

Generally Ambassador's readiness probe should be configured with the same settings as its liveness
probes.

## `AMBASSADOR_FAST_RECONFIGURE` and `AMBASSADOR_FAST_VALIDATION` Flags

These environment variables are feature flags that enable a higher performance implementation of the
code Ambassador uses to validate and generate envoy configuration. These will eventually be enabled
by default, but if you are experiencing performance problems you should try setting the values of
both of these flags to `"true"` and seeing if this helps.

## `AMBASSADOR_DRAIN_TIME`

The `AMBASSADOR_DRAIN_TIME` variable controls how much of a grace period Ambassador provides active
clients when reconfiguration happen. Its unit is seconds and it defaults to 600 (10 minutes). This
can impact memory usage because Ambassador needs to keep around old versions of its configuration
for the duration of the drain time.

## Unconstrained Mappings with Many Hosts

When working with a large number of `Host` resources, it's important to understand the impact of
unconstrained `Mapping`s. An unconstrained `Mapping` is one that is not restricted to a specific
`Host`. Such a `Mapping` will create a route for all of your `Host`s. If this is what you want then
it is the appropriate thing to do, however if you do not intend to do this, then you can end up with
many more routes than you had intended and this can adversely impact performance.

## Inspecting Ambassador Performance

Ambassador internally tracks a number of key performance indicators. You can inspect these via the
debug endpoint at `localhost:8877/debug`. Note that the `AMBASSADOR_FAST_RECONFIGURE` flag needs to
be set to `"true"` for this endpoint to be present:

```bash
$ kubectl exec -n ambassador -it ${POD} curl localhost:8877/debug
{
  "timers": {
    # These two timers track how long it takes to respond to liveness and readiness probes.
    "check_alive": "7, 45.411495ms/61.85999ms/81.358927ms",
    "check_ready": "7, 49.951304ms/61.976205ms/86.279038ms",

    # These two timers track how long we spend updating our in-memory snapshot when our Kubernetes
    # watches tell us something has changed.
    "consulUpdate": "0, 0s/0s/0s",
    "katesUpdate": "3382, 28.662µs/102.784µs/95.220222ms",

    # These timers tell us how long we spend notifying the sidecars if changed input. This
    # includes how long the sidecars take to process that input.
    "notifyWebhook:diagd": "2, 1.206967947s/1.3298432s/1.452718454s",
    "notifyWebhooks": "2, 1.207007216s/1.329901037s/1.452794859s",

    # This timer tells us how long we spend parsing annotations.
    "parseAnnotations": "2, 21.944µs/22.541µs/23.138µs",

    # This timer tells us how long we spend reconciling changes to consul inputs.
    "reconcileConsul": "2, 50.104µs/55.499µs/60.894µs",

    # This timer tells us how long we spend reconciling secrets related changes to ambassador
    # inputs.
    "reconcileSecrets": "2, 18.704µs/20.786µs/22.868µs"
  },
  "values": {
    "envoyReconfigs": {
      "times": [
        "2020-11-06T13:13:24.218707995-05:00",
        "2020-11-06T13:13:27.185754494-05:00",
        "2020-11-06T13:13:28.612279777-05:00"
      ],
      "staleCount": 2,
      "staleMax": 0,
      "synced": true
    },
    "memory": "39.73Gi of Unlimited (0%)"
  }
}
```
