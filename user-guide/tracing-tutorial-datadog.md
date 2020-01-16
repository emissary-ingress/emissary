# DataDog APM Tracing

In this tutorial, we'll configure Ambassador Edge Stack to initiate a trace on some sample requests, and use DataDog APM to visualize them.

## Before You Get Started

This tutorial assumes you have already followed the Ambassador Edge Stack [Getting Started](../getting-started) guide. If you haven't done that already, you should do that now.

After completing the Getting Started guide you will have a Kubernetes cluster running Ambassador Edge Stack and the Quote of the Moment service. Let's walk through adding tracing to this setup.

## 1. Configure the DataDog agent

You will need to configure the DataDog agent so that it uses a host-port and accepts non-local APM traffic, you can follow the DataDog [documentation](https://docs.datadoghq.com/agent/kubernetes/daemonset_setup/?tab=k8sfile#apm-and-distributed-tracing) on how to do this.

## 2. Configure Envoy JSON logging

DataDog APM can [correlate traces with logs](https://docs.datadoghq.com/tracing/advanced/connect_logs_and_traces/) if you propogate the current span and trace IDs with your logs.

When using JSON logging with Envoy, Ambassador Edge Stack will automatically append the `dd.trace_id` and `dd.span_id` properties to all logs so that correlation works:

```yaml
---
apiVersion: getambassador.io/v2
kind: Module
metadata:
  name: ambassador
config:
  envoy_log_type: json
```

## 3. Configure the TracingService

Next configure a TracingService that will write your traces using the DataDog tracing driver, as you want to write traces to your host-local DataDog agent you can use the `${HOST_IP}` interpolation to get the host IP address from the Ambassador Edge Stack containers environment.

```yaml
---
apiVersion: getambassador.io/v2
kind: TracingService
metadata:
  name: tracing
spec:
  service: "${HOST_IP}:8126"
  driver: datadog
  config:
    service_name: test
```

## 4. Generate some requests

Use `curl` to generate a few requests to an existing Ambassador Edge Stack mapping. You may need to perform many requests since only a subset of random requests are sampled and instrumented with traces.

```shell
$ curl -L $AMBASSADOR_IP/httpbin/ip
```

## 5. Test traces

Once you have made some requests you should be able to [view your traces](https://app.datadoghq.com/apm/traces) within a few minutes in the DataDog UI. If you would like more information on DataDog APM to learn about its features and benefits you can view the [documentation](https://docs.datadoghq.com/tracing/).

## More

For more details about configuring the external tracing service, read the documentation on [external tracing](../../reference/services/tracing-service).

