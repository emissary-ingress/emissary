# The `:8877/metrics` endpoint

> For an overview of other options for gathering statistics on
> Ambassador, see the [Statistics and Monitoring](../) overview.

Each Ambassador pod exposes statistics and metrics for that pod at
`http://{POD}:8877/metrics`.  The response is in the text-based
Prometheus [exposition format][].

[exposition format]: https://prometheus.io/docs/instrumenting/exposition_formats/

## Understanding the statistics

The Prometheus exposition format includes special "HELP" lines that
make the file self-documenting as to what specific statistics mean.

<!--

  TODO(lukeshu): Go in to more detail about Envoy's statistics; the
  discoverability of them in Envoy's docs is really bad.  The best
  thing to grep for in envoy.git is:

     git grep -E ', *(Gauge|Counter|Histogram) *,' docs

-->

- `envoy_*`: See the [Envoy documentation][`GET /stats/prometheus`].
- `ambassador_*` (new in 1.7.0):
  - `ambassador_edge_stack_*` (not present in Ambassador API Gateway):
    - `ambassador_edge_stack_go_*`: See [`promethus.NewGoCollector()`][].
    - `ambassador_edge_stack_promhttp_*` See [`promhttp.Handler()`][].
    - `ambassador_edge_stack_process_*`: See [`promethus.NewProcessCollector()`][]..
  - `ambassador_*_time_seconds` (for `*` = one of `aconf`, `diagnostics`, `econf`, `fetcher`, `ir`, or `reconfiguration`):
    Gauges of how long the various core operations take in the diagd
    process.
  - `ambassador_diagnostics_(errors|notices)`: The number of
    diagnostics errors and notices that would be shown in the
    diagnostics UI or the Edge Policy Console.
  - `ambassador_diagnostics_info`: [Info][`prometheus_client.Info`]
    about the Ambassador install; all information is presented in
    labels; the value of the Gauge is always "1".
  - `ambassador_process_*`: See [`prometheus_client.ProcessCollector`][].

[`GET /stats/prometheus`]: https://www.envoyproxy.io/docs/envoy/v1.15.0/operations/admin.html#get--stats-prometheus
[`prometheus.NewProcessCollector`]: https://godoc.org/github.com/prometheus/client_golang/prometheus#NewProcessCollector
[`prometheus.NewGoCollector`]: https://godoc.org/github.com/prometheus/client_golang/prometheus#NewGoCollector
[`promhttp.Handler()`]: https://godoc.org/github.com/prometheus/client_golang/prometheus/promhttp#Handler
[`prometheus_client.Info`]: https://github.com/prometheus/client_python#info
[`prometheus_client.ProcessCollector`]: https://github.com/prometheus/client_python#process-collector

## Polling the `:8877/metrics` endpoint with Prometheus

To scrape metrics directly, follow the instructions for [Monitoring
with Prometheus and Grafana](../../../../howtos/prometheus).

### Using Grafana to visualize statistics gathered by Prometheus

#### Sample dashboard

We provide a [sample Grafana dashboard](https://grafana.com/dashboards/13758)
that displays information collected by Prometheus from the
`:8877/metrics` endpoint.


#### Just Envoy information

![Screenshot of a Grafana dashboard that displays just information from Envoy](../../../../images/grafana.png)

[Alex Gervais][] has written a template [Ambassador dashboard for
Grafana][] that displays information collected by Prometheus either
from the `:8877/metrics` endpoint, or from [Envoy over
StatsD][envoy-statsd-prometheus].  Because it is designed to work with
the Envoy StatsD set up, it does not include any of the `ambassador_*`
statistics; because of this, we recommend using the other sample
dashboard above.

[Alex Gervais]: https://twitter.com/alex_gervais
[Ambassador dashboard for Grafana]: https://grafana.com/dashboards/4698
[envoy-statsd-prometheus]: ../envoy-statsd#using-prometheus-statsd-exporter-as-the-statsd-sink
