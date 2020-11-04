# The Ambassador Container

## Container Images

To give you flexibility and independence from a hosting platform's uptime, you can pull the `ambassador` and `aes` images from any of the following registries:
- `docker.io/datawire/`
- `quay.io/datawire/`
- `gcr.io/datawire/`

For an even more robust installation, consider using a [local registry as a pull through cache](https://docs.docker.com/registry/recipes/mirror/) or configure a [publicly accessible mirror](https://cloud.google.com/container-registry/docs/using-dockerhub-mirroring).

## Environment Variables

Use the following variables for the environment of your Ambassador container:

| Purpose                           | Variable                                    | Default value                                       | Value type                                                                    |
|-----------------------------------|---------------------------------------------|-----------------------------------------------------|-------------------------------------------------------------------------------|
| Core                              | `AMBASSADOR_ID`                             | `default`                                           | Plain string                                                                  |
| Core                              | `AMBASSADOR_NAMESPACE`                      | `default` ([^1])                                    | Kubernetes namespace                                                          |
| Core                              | `AMBASSADOR_SINGLE_NAMESPACE`               | Empty                                               | Boolean; non-empty=true, empty=false                                          |
| Core                              | `AMBASSADOR_ENVOY_BASE_ID`                  | `0`                                                 | Integer                                                                       |
| Core                              | `AMBASSADOR_FAST_VALIDATION`                | Empty                                               | EXPERIMENTAL -- Boolean; non-empty=true, empty=false                          |
| Core                              | `AMBASSADOR_FAST_RECONFIGURE`               | `false`                                             | EXPERIMENTAL -- Boolean; `true`=true, any other value=false                   |
| Core                              | `AMBASSADOR_UPDATE_MAPPING_STATUS`          | `false`                                             | Boolean; `true`=true, any other value=false                                   |
| Edge Stack                        | `AES_LOG_LEVEL`                             | `info`                                              | Log level (see below)                                                         |
| Primary Redis (L4)                | `REDIS_SOCKET_TYPE`                         | `tcp`                                               | Go network such as `tcp` or `unix`; see [Go `net.Dial`][]                     |
| Primary Redis (L4)                | `REDIS_URL`                                 | None, must be set explicitly                        | Go network address; for TCP this is a `host:port` pair; see [Go `net.Dial`][] |
| Primary Redis (L4)                | `REDIS_TLS_ENABLED`                         | `false`                                             | Boolean; [Go `strconv.ParseBool`][]                                           |
| Primary Redis (L4)                | `REDIS_TLS_INSECURE`                        | `false`                                             | Boolean; [Go `strconv.ParseBool`][]                                           |
| Primary Redis (auth)              | `REDIS_USERNAME`                            | Empty                                               | Plain string                                                                  |
| Primary Redis (auth)              | `REDIS_PASSWORD`                            | Empty                                               | Plain string                                                                  |
| Primary Redis (tune)              | `REDIS_POOL_SIZE`                           | `10`                                                | Integer                                                                       |
| Primary Redis (tune)              | `REDIS_PING_INTERVAL`                       | `10s`                                               | Duration; [Go `time.ParseDuration`][]                                         |
| Primary Redis (tune)              | `REDIS_TIMEOUT`                             | `0s`                                                | Duration; [Go `time.ParseDuration`][]                                         |
| Primary Redis (tune)              | `REDIS_SURGE_LIMIT_INTERVAL`                | `0s`                                                | Duration; [Go `time.ParseDuration`][]                                         |
| Primary Redis (tune)              | `REDIS_SURGE_LIMIT_AFTER`                   | The value of `REDIS_POOL_SIZE`                      | Integer                                                                       |
| Primary Redis (tune)              | `REDIS_SURGE_POOL_SIZE`                     | `0`                                                 | Integer                                                                       |
| Primary Redis (tune)              | `REDIS_SURGE_POOL_DRAIN_INTERVAL`           | `1m`                                                | Duration; [Go `time.ParseDuration`][]                                         |
| Per-Second RateLimit Redis        | `REDIS_PERSECOND`                           | `false`                                             | Boolean; [Go `strconv.ParseBool`][]                                           |
| Per-Second RateLimit Redis (L4)   | `REDIS_PERSECOND_SOCKET_TYPE`               | None, must be set explicitly (if `REDIS_PERSECOND`) | Go network such as `tcp` or `unix`; see [Go `net.Dial`][]                     |
| Per-Second RateLimit Redis (L4)   | `REDIS_PERSECOND_URL`                       | None, must be set explicitly (if `REDIS_PERSECOND`) | Go network address; for TCP this is a `host:port` pair; see [Go `net.Dial`][] |
| Per-Second RateLimit Redis (L4)   | `REDIS_PERSECOND_TLS_ENABLED`               | `false`                                             | Boolean; [Go `strconv.ParseBool`][]                                           |
| Per-Second RateLimit Redis (L4)   | `REDIS_PERSECOND_TLS_INSECURE`              | `false`                                             | Boolean; [Go `strconv.ParseBool`][]                                           |
| Per-Second RateLimit Redis (auth) | `REDIS_PERSECOND_USERNAME`                  | Empty                                               | Plain string                                                                  |
| Per-Second RateLimit Redis (auth) | `REDIS_PERSECOND_PASSWORD`                  | Empty                                               | Plain string                                                                  |
| Per-Second RateLimit Redis (tune) | `REDIS_PERSECOND_POOL_SIZE`                 | `10`                                                | Integer                                                                       |
| Per-Second RateLimit Redis (tune) | `REDIS_PERSECOND_PING_INTERVAL`             | `10s`                                               | Duration; [Go `time.ParseDuration`][]                                         |
| Per-Second RateLimit Redis (tune) | `REDIS_PERSECOND_TIMEOUT`                   | `0s`                                                | Duration; [Go `time.ParseDuration`][]                                         |
| Per-Second RateLimit Redis (tune) | `REDIS_PERSECOND_SURGE_LIMIT_INTERVAL`      | `0s`                                                | Duration; [Go `time.ParseDuration`][]                                         |
| Per-Second RateLimit Redis (tune) | `REDIS_PERSECOND_SURGE_LIMIT_AFTER`         | The value of `REDIS_PERSECOND_POOL_SIZE`            | Integer                                                                       |
| Per-Second RateLimit Redis (tune) | `REDIS_PERSECOND_SURGE_POOL_SIZE`           | `0`                                                 | Integer                                                                       |
| Per-Second RateLimit Redis (tune) | `REDIS_PERSECOND_SURGE_POOL_DRAIN_INTERVAL` | `1m`                                                | Duration; [Go `time.ParseDuration`][]                                         |
| RateLimit                         | `EXPIRATION_JITTER_MAX_SECONDS`             | `300`                                               | Integer                                                                       |
| RateLimit                         | `USE_STATSD`                                | `false`                                             | Boolean; [Go `strconv.ParseBool`][]                                           |
| RateLimit                         | `STATSD_HOST`                               | `localhost`                                         | Hostname                                                                      |
| RateLimit                         | `STATSD_PORT`                               | `8125`                                              | Integer                                                                       |
| RateLimit                         | `GOSTATS_FLUSH_INTERVAL_SECONDS`            | `5`                                                 | Integer                                                                       |
| Developer Portal                  | `AMBASSADOR_URL`                            | `https://api.example.com`                           | URL                                                                           |
| Developer Portal                  | `DEVPORTAL_CONTENT_URL`                     | `https://github.com/datawire/devportal-content`     | git-remote URL                                                                |
| Developer Portal                  | `DEVPORTAL_CONTENT_DIR`                     | `/`                                                 | Rooted Git directory                                                          |
| Developer Portal                  | `DEVPORTAL_CONTENT_BRANCH`                  | `master`                                            | Git branch name                                                               |
| Developer Portal                  | `POLL_EVERY_SECS`                           | `60`                                                | Integer                                                                       |
| Envoy                             | `STATSD_ENABLED`                            | `false`                                             | Boolean; Python `value.lower() == "true"`                                     |
| Envoy                             | `DOGSTATSD`                                 | `false`                                             | Boolean; Python `value.lower() == "true"`                                     |

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

#### Redis layer 4 connectivity (L4)

- `SOCKET_TYPE` and `URL` are the Go network name and Go network
  address to dial to talk to Redis; see [Go `net.Dial`][].  Note that
  when using a port name instead of a port number, the name refers a
  well-known port name in the container's `/etc/services`, and **not**
  to a Kubernetes port name.  For `REDIS_URL` (but not
  `REDIS_PERSECOND_URL`), not setting a value disables Ambassador Edge
  Stack features that require Redis.
- `TLS_ENABLED` (new in 1.5.0) specifies whether to use TLS when
  talking to Redis.
- `TLS_INSECURE` (new in 1.5.0) specifies whether to skip certificate
  verification when using TLS to talk to Redis.  Alternatively,
  consider [installing Redis' self-signed certificate in to the
  Ambassador Edge Stack
  container](../../using/filters/#installing-self-signed-certificates)
  in order to leave certificate verification on.

#### Redis authentication (auth)

- If `PASSWORD` (new in 1.5.0) is non-empty, then it is used to `AUTH`
  to Redis immediately after the connection is established.
- If `USERNAME` (new in 1.5.0) is set, then that username is used with
  the password to log in as that user in the [Redis 6 ACL][].  It is
  invalid to set a username without setting a password.  It is invalid
  to set a username with Redis 5 or lower.

#### Redis performance tuning (tune)

- `POOL_SIZE` is the number of connections to keep around when idle.
  The total number of connections may go lower than this if there are
  errors.  The total number of connections may go higher than this
  during a load surge.
- `PING_INTERVAL` (new in 1.6.0; changed in 1.7.0) Of the idle
  connections in the normal pool (not extra connections created for a
  load surge), Ambassador will `PING` one of them every
  `PING_INTERVAL÷POOL_SIZE`; each connection will on average be
  `PING`ed every `PING_INTERVAL`.  (Backward incompatibility: in 1.6.x
  Ambassador did not divide by `POOL_SIZE`; which itself was
  backward-incompatible from the pre-1.6.0 behavior of using
  `10s÷POOL_SIZE`; 1.7.0 defaults to the pre-1.6.0 behavior.)
  (Backward incompatibility: in 1.7.0 this changed from an integer
  number of seconds to a duration-string; enabling sub-second values.)
- `TIMEOUT` (new in 1.6.0; changed in 1.7.0) sets 4 different timeouts:

   1. `(*net.Dialer).Timeout` for establishing connections
   2. `(*redis.Client).ReadTimeout` for reading a single complete response
   3. `(*redis.Client).WriteTimeout` for writing a single complete request
   4. The timeout when waiting for a connection to become available
      from the pool (not including the dial time, which is timed out
      separately) (since 1.7.0)

  A value of "0" means "no timeout".

  (Backward incompatibility: in 1.7.0 this was renamed from
  `IO_TIMEOUT` to `TIMEOUT`; changed from an integer number of seconds
  to a duration-string, enabling sub-second values; and the default
  value changed from "10s" to "0", defaulting to the pre-1.6.0
  behavior.)

- `SURGE_LIMIT_INTERVAL` (new in 1.7.0) During a load surge, if the
  pool is depleted, then Ambassador may create new connections to
  Redis in order to fulfill demand, at a maximum rate of one new
  connection per `SURGE_LIMIT_INTERVAL`.  A value of "0" (the default)
  means "allow new connections to be created as fast as necessary.
  (Backward incompatibility: in 1.6.x this was a non-configurable
  "1s"; in 1.7.0 the default value is "0", defaulting to the pre-1.6.0
  behavior.)  The total number of connections that Ambassador can
  surge to is unbounded.

- `SURGE_LIMIT_AFTER` (new in 1.7.0) is how many connections *after*
  the normal pool is depleted can be created before
  `SURGE_LIMIT_INTERVAL` kicks in; the first
  `POOL_SIZE+SURGE_LIMIT_AFTER` connections are allowed to be created
  as fast as necessary.  This setting has no effect if
  `SURGE_LIMIT_INTERVAL` is 0.

- `SURGE_POOL_SIZE` (new in 1.6.0; changed in 1.7.0) Normally during a
  surge, excess connections beyond `POOL_SIZE` are closed immediately
  after they are done being used, instead of being returned to a pool.
  `SURGE_POOL_SIZE` configures a "reserve" pool of size
  `SURGE_POOL_SIZE` for excess connections created during a surge.
  Excess connections beyond `POOL_SIZE+SURGE_POOL_SIZE` will still be
  closed immediately after use.  (Backward incompatibility: in 1.7.0
  this was renamed from `POOL_MAX_SIZE` to `SURGE_POOL_SIZE`; and the
  default value was changed from "20" to "0", now defaulting to the
  pre-1.6.0 behavior.)

- `SURGE_POOL_DRAIN_INTERVAL` (new in 1.7.0) is how quickly to drain
  connections from the surge pool after a surge is over; connections
  are closed at a rate of one connection per
  `SURGE_POOL_DRAIN_INTERVAL`.  This setting has no effect if
  `SURGE_POOL_SIZE` is 0.

## Port Assignments

The Ambassador Edge Stack uses the following ports to listen for HTTP/HTTPS traffic automatically via TCP:

| Port | Process | Function                                                |
|------|---------|---------------------------------------------------------|
| 8001 | envoy   | Internal stats, logging, etc.; not exposed outside pod  |
| 8002 | watt    | Internal watt snapshot access; not exposed outside pod  |
| 8003 | ambex   | Internal ambex snapshot access; not exposed outside pod |
| 8004 | diagd   | Internal `diagd` access when `AMBASSADOR_FAST_RECONFIGURE` is set; not exposed outside pod |
| 8080 | envoy   | Default HTTP service port                               |
| 8443 | envoy   | Default HTTPS service port                              |
| 8877 | diagd   | Direct access to diagnostics UI; provided by `busyambassador entrypoint` when `AMBASSADOR_FAST_RECONFIGURE` is set |

[^1]: This may change in a future release to reflect the Pods's
      namespace if deployed to a namespace other than `default`.
      https://github.com/datawire/ambassador/issues/1583

[Go `net.Dial`]: https://golang.org/pkg/net/#Dial
[Go `strconv.ParseBool`]: https://golang.org/pkg/strconv/#ParseBool
[Go `time.ParseDuration`]: https://golang.org/pkg/strconv/#ParseDuration
[Redis 6 ACL]: https://redis.io/topics/acl
