# The Ambassador Container

## Container Images

To give you flexibility and independence from a hosting platform's uptime, you can pull the `ambassador` and `aes` images from any of the following registries:
- `docker.io/datawire/`
- `quay.io/datawire/`
- `gcr.io/datawire/`

For an even more robust installation, consider using a [local registry as a pull through cache](https://docs.docker.com/registry/recipes/mirror/) or configure a [publicly accessible mirror](https://cloud.google.com/container-registry/docs/using-dockerhub-mirroring).

## Environment Variables

Use the following variables for the environment of your Ambassador container:

| Purpose                           | Variable                           | Default value                                       | Value type                                                                    |
|-----------------------------------|------------------------------------|-----------------------------------------------------|-------------------------------------------------------------------------------|
| Ambassador                        | `AMBASSADOR_ID`                    | `default`                                           | Plain string                                                                  |
| Ambassador                        | `AMBASSADOR_NAMESPACE`             | `default` ([^1])                                    | Kubernetes namespace                                                          |
| Ambassador                        | `AMBASSADOR_SINGLE_NAMESPACE`      | Empty                                               | Boolean; non-empty=true, empty=false                                          |
| Ambassador                        | `AMBASSADOR_ENVOY_BASE_ID`         | `0`                                                 | Integer                                                                       |
| Ambassador                        | `AMBASSADOR_FAST_VALIDATION`       | Empty                                               | EXPERIMENTAL -- Boolean; non-empty=true, empty=false                          |
| Ambassador                        | `AMBASSADOR_UPDATE_MAPPING_STATUS` | `false`                                             | Boolean; `true`=true, any other value=false                                   |
| Ambassador Edge Stack             | `AES_LOG_LEVEL`                    | `info`                                              | Log level (see below)                                                         |
| Primary Redis (L4)                | `REDIS_SOCKET_TYPE`                | None, must be set explicitly                        | Go network such as `tcp` or `unix`; see [Go `net.Dial`][]                     |
| Primary Redis (L4)                | `REDIS_URL`                        | None, must be set explicitly                        | Go network address; for TCP this is a `host:port` pair; see [Go `net.Dial`][] |
| Primary Redis (L4)                | `REDIS_TLS_ENABLED`                | `false`                                             | Boolean; [Go `strconv.ParseBool`][]                                           |
| Primary Redis (L4)                | `REDIS_TLS_INSECURE`               | `false`                                             | Boolean; [Go `strconv.ParseBool`][]                                           |
| Primary Redis (auth)              | `REDIS_USERNAME`                   | Empty                                               | Plain string                                                                  |
| Primary Redis (auth)              | `REDIS_PASSWORD`                   | Empty                                               | Plain string                                                                  |
| Primary Redis (tune)              | `REDIS_POOL_SIZE`                  | `10`                                                | Integer                                                                       |
| Primary Redis (tune)              | `REDIS_POOL_MAX_SIZE`              | `20`                                                | Integer                                                                       |
| Primary Redis (tune)              | `REDIS_PING_INTERVAL`              | `10`                                                | Integer (seconds)                                                             |
| Primary Redis (tune)              | `REDIS_IO_TIMEOUT`                 | `10`                                                | Integer (seconds)                                                             |
| Per-Second RateLimit Redis        | `REDIS_PERSECOND`                  | `false`                                             | Boolean; [Go `strconv.ParseBool`][]                                           |
| Per-Second RateLimit Redis (L4)   | `REDIS_PERSECOND_SOCKET_TYPE`      | None, must be set explicitly (if `REDIS_PERSECOND`) | Go network such as `tcp` or `unix`; see [Go `net.Dial`][]                     |
| Per-Second RateLimit Redis (L4)   | `REDIS_PERSECOND_URL`              | None, must be set explicitly (if `REDIS_PERSECOND`) | Go network address; for TCP this is a `host:port` pair; see [Go `net.Dial`][] |
| Per-Second RateLimit Redis (L4)   | `REDIS_PERSECOND_TLS_ENABLED`      | `false`                                             | Boolean; [Go `strconv.ParseBool`][]                                           |
| Per-Second RateLimit Redis (L4)   | `REDIS_PERSECOND_TLS_INSECURE`     | `false`                                             | Boolean; [Go `strconv.ParseBool`][]                                           |
| Per-Second RateLimit Redis (auth) | `REDIS_PERSECOND_USERNAME`         | Empty                                               | Plain string                                                                  |
| Per-Second RateLimit Redis (auth) | `REDIS_PERSECOND_PASSWORD`         | Empty                                               | Plain string                                                                  |
| Per-Second RateLimit Redis (tune) | `REDIS_PERSECOND_POOL_SIZE`        | `10`                                                | Integer                                                                       |
| Per-Second RateLimit Redis (tune) | `REDIS_PERSECOND_POOL_MAX_SIZE`    | `20`                                                | Integer                                                                       |
| Per-Second RateLimit Redis (tune) | `REDIS_PERSECOND_PING_INTERVAL`    | `10`                                                | Integer (seconds)                                                             |
| Per-Second RateLimit Redis (tune) | `REDIS_PERSECOND_IO_TIMEOUT`       | `10`                                                | Integer (seconds)                                                             |
| RateLimit                         | `EXPIRATION_JITTER_MAX_SECONDS`    | `300`                                               | Integer                                                                       |
| RateLimit                         | `USE_STATSD`                       | `false`                                             | Boolean; [Go `strconv.ParseBool`][]                                           |
| RateLimit                         | `STATSD_HOST`                      | `localhost`                                         | Hostname                                                                      |
| RateLimit                         | `STATSD_PORT`                      | `8125`                                              | Integer                                                                       |
| RateLimit                         | `GOSTATS_FLUSH_INTERVAL_SECONDS`   | `5`                                                 | Integer                                                                       |
| Developer Portal                  | `AMBASSADOR_URL`                   | `https://api.example.com`                           | URL                                                                           |
| Developer Portal                  | `DEVPORTAL_CONTENT_URL`            | `https://github.com/datawire/devportal-content`     | git-remote URL                                                                |
| Developer Portal                  | `DEVPORTAL_CONTENT_DIR`            | `/`                                                 | Rooted Git directory                                                          |
| Developer Portal                  | `DEVPORTAL_CONTENT_BRANCH`         | `master`                                            | Git branch name                                                               |
| Developer Portal                  | `POLL_EVERY_SECS`                  | `60`                                                | Integer                                                                       |

Log level names are case-insensitive.  From least verbose to most
verbose, valid log levels are `error`, `warn`/`warning`, `info`,
`debug`, and `trace`.

### Redis

The Ambassador Edge Stack make use of Redis for several purposes.  By
default, all components of the Ambassador Edge Stack share a Redis
connection pool.  If `REDIS_PERSECOND` is true, a second Redis
connection pool is created (to a potentially different Redis instance)
that is only used for per-second RateLimits; this second connection
pool is configured by the `REDIS_PERSECOND_*` variables rather than
the usual `REDIS_*` variables.

#### Redis layer 4 connectivity

- `SOCKET_TYPE` and `URL` are the Go network name and Go network
  address to dial to talk to Redis; see [Go `net.Dial`][].  Note that
  when using a port name instead of a port number, the name refers a
  well-known port name in the container's `/etc/services`, and **not**
  to a Kubernetes port name.
- `TLS_ENABLED` (new in 1.5.0) specifies whether to use TLS when
  talking to Redis.
- `TLS_INSECURE` (new in 1.5.0) specifies whether to skip certificate
  verification when using TLS to talk to Redis.  Alternatively,
  consider [installing Redis' self-signed certificate in to the
  Ambassador Edge Stack
  container](../../using/filters/#installing-self-signed-certificates)
  in order to leave certificate verification on.

#### Redis authentication

- If `PASSWORD` (new in 1.5.0) is non-empty, then it is used to `AUTH`
  to Redis immediately after the connection is established.
- If `USERNAME` (new in 1.5.0) is set, then that username is used with
  the password to log in as that user in the [Redis 6 ACL].  It is
  invalid to set a username without setting a password.  It is invalid
  to set a username with Redis 5 or lower.

#### Redis performance tuning

- `POOL_SIZE` is the number of connections to keep around when idle.
  The total number of connections may go lower than this if there are
  errors.  The total number of connections may go higher than this
  during a load surge.
- `PING_INTERVAL` (new in 1.6.0) Of the idle connections in the normal
  pool (not extra connections created for a load surge), Ambassador
  will `PING` one of them every `PING_INTERVAL` seconds; each
  connection will on average be `PING`ed every
  `PING_INTERVAL×POOL_SIZE` seconds; increasing `POOL_SIZE` without
  reducing `PING_INTERVAL` will increase the amount of time between
  `PING`s on a given connection.  (Backward incompatibility: prior to
  the introduction of this setting in 1.6.0, it behaved as if
  `PING_INTERVAL=10s÷POOL_SIZE`.)
- `IO_TIMEOUT` sets 3 different timeouts:
   1. `(*net.Dialer).Timeout` for establishing connections
   2. `(*redis.Client).ReadTimeout` for reading a single complete response
   3. `(*redis.Client).WriteTimeout` for writing a single complete request

During a load surge, if the pool is depleted, Ambassador allows new
connections to be created as fast as necessary for the first
`POOL_SIZE` connections; once the number of connections reaches
`2×POOL_SIZE` it is limited to creating only 1 new connection per
second.  (Backward incompatibility: prior to 1.6.0 it never limited
the creation of new connections during a surge.)

- `POOL_MAX_SIZE` (new in 1.6.0) During a load surge, instead of
  closing connections immediately after use, they are placed in to a
  "reserve" pool of size `POOL_MAX_SIZE`.  (Backward incompatibility:
  prior to 1.6.0 there was no reserve pool.)  Excess connections
  beyond `POOL_SIZE+POOL_MAX_SIZE` will still be closed immediately
  after use.  Connections in the reserve pool are drained at a rate of
  1 connection per minute.

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
[Redis 6 ACL]: https://redis.io/topics/acl
