---
layout: doc
weight: 3.1
title: "Statistics and Monitoring"
categories: user-guide
---
Ambassador is an API Gateway for microservices built on [Envoy](https://lyft.github.io/envoy/). A key feature of Envoy is the observability it enables by exposing a multitude of statistics about its own operations. Ambassador makes it easy to direct this information to a statistics and monitoring tool of your choice.

As an example, for a given service `usersvc`, here are some interesting statistics to investigate:

- `envoy.cluster.usersvc_cluster.upstream_rq_total` is the total number of requests that `usersvc` has received via Ambassador. The rate of change of this value is one basic measure of service utilization, i.e. requests per second.
- `envoy.cluster.usersvc_cluster.upstream_rq_2xx` is the total number of requests to which `usersvc` responded with an HTTP response indicating success. This value divided by the prior one, taken on an rolling window basis, represents the recent success rate of the service. There are corresponding `4xx` and `5xx` counters that can help clarify the nature of unsuccessful requests.
- `envoy.cluster.usersvc_cluster.upstream_rq_time` is a StatsD timer that tracks the latency in milliseconds of `usersvc` from Ambassador's perspective. StatsD timers include information about means, standard deviations, and decile values.

Statistics are exposed via the ubiquitous and well-tested [StatsD](https://github.com/etsy/statsd) protocol. Ambassador automatically sends statistics information to a Kubernetes service called `statsd-sink` using typical StatsD protocol settings, UDP to port 8125. We have included a few example configurations in the [statsd-sink](https://github.com/datawire/ambassador/tree/master/statsd-sink) subdirectory to help you get started.


### Graphite

Applying the `statsd-sink/graphite/graphite-statsd-sink.yaml` file will launch a service and deployment in your cluster that sets up [Graphite](http://graphite.readthedocs.org/) and its related infrastructure. Once launched, you can access Graphite's web interface at `http://statsd-sink/` from within the cluster, or use `kubectl port-forward` to gain access from your local machine.


### Prometheus

Applying the `statsd-sink/prometheus/prom-statsd-sink.yaml` file will launch a service and deployment of the Prometheus StatsD Exporter in your cluster. The deployment is based on Docker image defined in the same directory. You will need to set up a Prometheus target to read from `statsd-sink` on port 9102.

If you don't already have a Prometheus setup, you can spin up an example in your cluster using [Helm](https://github.com/kubernetes/helm):

    $ helm init
    $ helm install stable/prometheus --name prom -f helm-prom-config.yaml

The supplied configuration file `helm-prom-config.yaml` in the same directory is configured with `statsd-sink` as a target. Once Prometheus is running, use `kubectl port-forward prom-prometheus-server-[...] 9090` to setup port forwarding. Then you can access the Prometheus web interface on `http://localhost:9090/`.


### Datadog

The `statsd-sink/datadog/dd-statsd-sink.yaml` file, once edited to contain your Datadog API key, allows you to launch a service and deployment of Datadog's DogStatsD agent. It will automatically forward Ambassador stats to your Datadog account.
