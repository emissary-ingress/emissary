# The Ambassador Container

## Environment Variables

Use the following variables for the environment of your Ambassador container:

| Purpose                    | Variable                         | Default value                                       | Value type                                                                    |
|----------------------------|----------------------------------|-----------------------------------------------------|-------------------------------------------------------------------------------|
| Ambassador                 | `AMBASSADOR_ID`                  | `default`                                           | Plain string                                                                  |
| Ambassador                 | `AMBASSADOR_NAMESPACE`           | `default` ([^1])                                    | Kubernetes namespace                                                          |
| Ambassador                 | `AMBASSADOR_SINGLE_NAMESPACE`    | Empty                                               | Boolean; non-empty=true, empty=false                                          |
| Ambassador Edge Stack      | `AES_LOG_LEVEL`                  | `info`                                              | Log level (see below)                                                         |
| Primary Redis              | `REDIS_POOL_SIZE`                | `10`                                                | Integer                                                                       |
| Primary Redis              | `REDIS_SOCKET_TYPE`              | None, must be set explicitly                        | Go network such as `tcp` or `unix`; see [Go `net.Dial`][]                     |
| Primary Redis              | `REDIS_URL`                      | None, must be set explicitly                        | Go network address; for TCP this is a `host:port` pair; see [Go `net.Dial`][] |
| Per-Second RateLimit Redis | `REDIS_PERSECOND`                | `false`                                             | Boolean; [Go `strconv.ParseBool`][]                                           |
| Per-Second RateLimit Redis | `REDIS_PERSECOND_POOL_SIZE`      | `10`                                                | Integer                                                                       |
| Per-Second RateLimit Redis | `REDIS_PERSECOND_SOCKET_TYPE`    | None, must be set explicitly (if `REDIS_PERSECOND`) | Go network such as `tcp` or `unix`; see [Go `net.Dial`][]                     |
| Per-Second RateLimit Redis | `REDIS_PERSECOND_URL`            | None, must be set explicitly (if `REDIS_PERSECOND`) | Go network address; for TCP this is a `host:port` pair; see [Go `net.Dial`][] |
| RateLimit                  | `EXPIRATION_JITTER_MAX_SECONDS`  | `300`                                               | Integer                                                                       |
| RateLimit                  | `USE_STATSD`                     | `false`                                             | Boolean; [Go `strconv.ParseBool`][]                                           |
| RateLimit                  | `STATSD_HOST`                    | `localhost`                                         | Hostname                                                                      |
| RateLimit                  | `STATSD_PORT`                    | `8125`                                              | Integer                                                                       |
| RateLimit                  | `GOSTATS_FLUSH_INTERVAL_SECONDS` | `5`                                                 | Integer                                                                       |
| Developer Portal           | `AMBASSADOR_URL`                 | `https://api.example.com`                           | URL                                                                           |
| Developer Portal           | `DEVPORTAL_CONTENT_URL`          | `https://github.com/datawire/devportal-content`     | git-remote URL                                                                |
| Developer Portal           | `DEVPORTAL_CONTENT_DIR`          | `/`                                                 | Rooted Git directory                                                          |
| Developer Portal           | `DEVPORTAL_CONTENT_BRANCH`       | `master`                                            | Git branch name                                                               |
| Developer Portal           | `POLL_EVERY_SECS`                | `60`                                                | Integer                                                                       |

Log level names are case-insensitive.  From least verbose to most
verbose, valid log levels are `error`, `warn`/`warning`, `info`,
`debug`, and `trace`.

### Redis

The Ambassador Edge Stack make use of Redis for several purposes.  By
default, all components of the Ambassador Edge Stack share a Redis
connection pool; there will be a total of up to `REDIS_POOL_SIZE`
connections to Redis.  If `REDIS_PERSECOND` is true, a second Redis
connection pool is created (to a potentially different Redis instance)
that is only used for per-second RateLimits; this second connection
pool is configured by the `REDIS_PERSECOND_*` variables rather than
the usual `REDIS_*` variables.

Note that when using a port name instead of a port number in a Go
network address (as as in `REDIS_URL` or `REDIS_PERSECOND_URL`), the
name refers a well-known port name in the container's `/etc/services`,
and **not** to a Kubernetes port name.

## Port Assignments

The Ambassador Edge Stack uses the following ports to listen for HTTP/HTTPS traffic automatically via TCP:

| Port | Process | Function                                                |
|------|---------|---------------------------------------------------------|
| 8001 | envoy   | Internal stats, logging, etc.; not exposed outside pod  |
| 8002 | watt    | Internal watt snapshot access; not exposed outside pod  |
| 8003 | ambex   | Internal ambex snapshot access; not exposed outside pod |
| 8080 | envoy   | Default HTTP service port                               |
| 8443 | envoy   | Default HTTPS service port                              |

[^1]: This may change in a future release to reflect the Pods's
      namespace if deployed to a namespace other than `default`.
      https://github.com/datawire/ambassador/issues/1583

[Go `net.Dial`]: https://golang.org/pkg/net/#Dial
[Go `strconv.ParseBool`]: https://golang.org/pkg/strconv/#ParseBool
