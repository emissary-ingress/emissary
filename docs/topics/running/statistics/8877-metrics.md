# The `:8877/metrics` endpoint

> For an overview of other options for gathering statistics on
> Ambassador, see the [Statistics and Monitoring](../) overview.

Each Ambassador pod exposes statistics and metrics for that pod at
`http://{POD}:8877/metrics`.

You can use the Envoy `/metrics` endpoint to scrape states and metrics
directly, so you don't need to configure your Ambassador Edge Stack to
output statistics to another tool, such as StatsD.

## Polling the `:8877/metrics` endpoint with Prometheus

To scrape metrics directly, follow the instructions for [Monitoring
with Prometheus and Grafana](../../../../howtos/prometheus).

### Using Grafana to visualize statistics gathered by Prometheus

![Grafana dashboard](../../../../images/grafana.png)

If you're using Grafana, [Alex Gervais][] has written a template
[Ambassador dashboard for Grafana][] that works with either the
metrics exposed by [the `:8877/metrics` endpoint], or by [Envoy over
StatsD][envoy-statsd-prometheus].

[Alex Gervais]: https://twitter.com/alex_gervais
[Ambassador dashboard for Grafana]: https://grafana.com/dashboards/4698
[envoy-statsd-prometheus]: ../envoy-statsd#using-prometheus-statsd-exporter-as-the-statsd-sink
