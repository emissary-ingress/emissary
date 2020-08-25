# Statistics and Monitoring

Ambassador collects many statistics internally, and makes it easy to
direct this information to a statistics and monitoring tool of your
choice.  As an example, for a given service `usersvc`, here are some
interesting statistics to investigate:

- `envoy.cluster.usersvc_cluster.upstream_rq_total` is the total
  number of requests that `usersvc` has received via Ambassador Edge
  Stack.  The rate of change of this value is one basic measure of
  service utilization, i.e. requests per second.
- `envoy.cluster.usersvc_cluster.upstream_rq_2xx` is the total number
  of requests to which `usersvc` responded with an HTTP response
  indicating success.  This value divided by the prior one, taken on
  an rolling window basis, represents the recent success rate of the
  service.  There are corresponding `4xx` and `5xx` counters that can
  help clarify the nature of unsuccessful requests.
- `envoy.cluster.usersvc_cluster.upstream_rq_time` is a StatsD timer
  that tracks the latency in milliseconds of `usersvc` from Ambassador
  Edge Stack's perspective.  StatsD timers include information about
  means, standard deviations, and decile values.

There are several ways to get different statistics out of Ambassador:

- [The `:8877/metrics` endpoint](./8877-metrics) can be polled for
  aggregated statistics (in a Prometheus-compatible format).  This is
  our recommended method.
- Ambassador can push [Envoy statistics](./envoy-statsd) over the
  StatsD or DogStatsD protocol.
- Ambassador Edge Stack can push [RateLimiting
  statistics](../../environment) over the StatsD protocol.
