# Statistics and Monitoring

Ambassador is an API Gateway for microservices built on [Envoy](https://lyft.github.io/envoy/). A key feature of Envoy is the observability it enables by exposing a multitude of statistics about its own operations. Ambassador makes it easy to direct this information to a statistics and monitoring tool of your choice.

As an example, for a given service `usersvc`, here are some interesting statistics to investigate:

- `envoy.cluster.usersvc_cluster.upstream_rq_total` is the total number of requests that `usersvc` has received via Ambassador. The rate of change of this value is one basic measure of service utilization, i.e. requests per second.
- `envoy.cluster.usersvc_cluster.upstream_rq_2xx` is the total number of requests to which `usersvc` responded with an HTTP response indicating success. This value divided by the prior one, taken on an rolling window basis, represents the recent success rate of the service. There are corresponding `4xx` and `5xx` counters that can help clarify the nature of unsuccessful requests.
- `envoy.cluster.usersvc_cluster.upstream_rq_time` is a StatsD timer that tracks the latency in milliseconds of `usersvc` from Ambassador's perspective. StatsD timers include information about means, standard deviations, and decile values.

Statistics are exposed via the ubiquitous and well-tested [StatsD](https://github.com/etsy/statsd) protocol. Ambassador automatically sends statistics information to a Kubernetes service called `statsd-sink` using typical StatsD protocol settings, UDP to port 8125. We have included a few example configurations in the [statsd-sink](https://github.com/datawire/ambassador/tree/master/statsd-sink) subdirectory to help you get started. Clone the repository to get local, editable copies.


## Graphite

[Graphite](http://graphite.readthedocs.org/) is a web-based realtime graphing system. Spin up an example Graphite setup:

    kubectl apply -f statsd-sink/graphite/graphite-statsd-sink.yaml

This sets up the `statsd-sink` service and a deployment that contains Graphite and its related infrastructure. Graphite's web interface is available at `http://statsd-sink/` from within the cluster. Use port forwarding to access the interface from your local machine:

    SINKPOD=$(kubectl get pod -l service=statsd-sink -o jsonpath="{.items[0].metadata.name}")
    kubectl port-forward $SINKPOD 8080:80

This sets up Graphite access at `http://localhost:8080/`.


## Prometheus

[Prometheus](https://prometheus.io/) is an open-source monitoring and alerting system. If you already use Prometheus, use the sample StatsD Exporter YAML file to get started:

    kubectl apply -f statsd-sink/prometheus/prom-statsd-sink.yaml

This sets up the `statsd-sink` service and a deployment of the Prometheus StatsD Exporter in your cluster. The deployment is based on the Docker image defined in the same directory. Add a Prometheus target to read from `statsd-sink` on port 9102 to complete the Prometheus configuration.

If you don't already have a Prometheus setup, spin up an example in your cluster using [Helm](https://github.com/kubernetes/helm):

    helm install stable/prometheus --name prom -f statsd-sink/prometheus/helm-prom-config.yaml

The supplied configuration file `helm-prom-config.yaml` includes `statsd-sink` as a Prometheus target. Once Prometheus is running, set up port forwarding:

    PROMPOD=$(kubectl get pod -l service=prom-prometheus-server -o jsonpath="{.items[0].metadata.name}")
    kubectl port-forward $PROMPOD 9090

Now you can access the Prometheus web interface on `http://localhost:9090/`.


## Datadog

If you are a user of the [Datadog](https://www.datadoghq.com/) monitoring system, pulling in Ambassador statistics is very easy. Replace the sample API key in the YAML file with your own, then launch the DogStatsD agent:

    kubectl apply -f statsd-sink/datadog/dd-statsd-sink.yaml

This sets up the `statsd-sink` service and a deployment of the DogStatsD agent that automatically forwards Ambassador stats to your Datadog account.
