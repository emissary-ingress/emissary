# Environment variables for the Ambassador Edge Stack container


| Purpose&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;  | Variable | Default  | Value type &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;   |
|----------------------------------	|---------------------------------------------------	|-------------------------------------------------------------------------------	|------------------------------------------------------	|
| Ambassador	| `AMBASSADOR_ID`	| `default`	| Plain string	|
| Ambassador	| `AMBASSADOR_NAMESPACE`	| `default` ([^1])	| Kubernetes namespace	|
| Ambassador	| `AMBASSADOR_SINGLE_NAMESPACE`	| empty	| Boolean; non-empty=true, empty=false	|
| Ambassador general purpose	| `APP_LOG_LEVEL`	| `info`	| Log level	|
| Developer Portal	| `AMBASSADOR_ADMIN_URL`	| `http://127.0.0.1:8877`	| URL	|
| Developer Portal	| `AMBASSADOR_INTERNAL_URL`	| `https://127.0.0.1:8443`	| URL	|
| Developer Portal	| `AMBASSADOR_URL`	| `https://api.example.com`	| URL	|
| Developer Portal	| `APRO_DEVPORTAL_CONTENT_URL`	| `https://github.com/datawire/devportal-content`	| git-remote URL	|
| Developer Portal	| `POLL_EVERY_SECS`	| `60`	| Integer	|
| Filter	| `APRO_KEYPAIR_SECRET_NAME`	| `ambassador-pro-keypair`	| Kubernetes name	|
| Filter	| `APRO_KEYPAIR_SECRET_NAMESPACE`	| use the value of `AMBASSADOR_NAMESPACE`	| Kubernetes namespace	|
| Filter gRPC, RateLimit gRPC, health HTTP, debug HTTP	| `APRO_HTTP_PORT`	| `8500`	| TCP port number or name	|
| Filter, RateLimit	| `REDIS_POOL_SIZE`	| `10`	| Integer	|
| Filter, RateLimit	| `REDIS_SOCKET_TYPE`	| None, must be set manually	| Go network such as `tcp` or `unix`; see [Go `net.Dial`][]	|
| Filter, RateLimit	| `REDIS_URL`	| None, must be set manually	| Go network address; for TCP this is a `host:port` pair; see [Go `net.Dial`][]	|
| RateLimit	| `REDIS_PERSECOND`	| `false`	| Boolean; [Go `strconv.ParseBool`][]	|
| RateLimit	| `REDIS_PERSECOND_SOCKET_TYPE`	| None, must be set manually (if `REDIS_PERSECOND`)	| Go network such as `tcp` or `unix`; see [Go `net.Dial`][]	|
| RateLimit	| `REDIS_PERSECOND_POOL_SIZE`	| None, must be set manually (if `REDIS_PERSECOND`)	| Go network address; for TCP this is a `host:port` pair; see [Go `net.Dial`][]	|
| RateLimit	| `EXPIRATION_JITTER_MAX_SECONDS`	| `300`	| Integer	|
| RateLimit	| `USE_STATSD`	| `false`	| Boolean; [Go `strconv.ParseBool`][]	|
| RateLimit	| `STATSD_HOST`	| `localhost`	| Hostname	|
| RateLimit	| `STATSD_PORT`	| `8125`	| Integer	|
| RateLimit	| `GOSTATS_FLUSH_INTERVAL_SECONDS`	| `5`	| Integer	|
<!--

  Intentionally omit `RLS_RUNTIME_DIR` from the above table; it exists
  for development purposes and isn't meant to be set by end users.

-->

Port names are well-known port names that are recognized by
`/etc/services`, they are *not* Kubernetes port names.

Log levels are case-insensitive. From least verbose to most verbose,
valid log levels are `error`, `warn`/`warning`, `info`, `debug`, and
`trace`.

The AuthService and the RateLimitService share a Redis connection
pool; there will be up to `REDIS_POOL_SIZE` connections to Redis.

If `REDIS_PERSECOND` is true, a second Redis connection pool is
created (to a potentially different Redis instance) that is only used
for per-second RateLimits.

If the `APRO_KEYPAIR_SECRET_NAME`/`APRO_KEYPAIR_SECRET_NAMESPACE`
Kubernetes secret does not already exist when Ambassador Edge Stack starts,
it will be automatically created; which obviously requires permission
in the ClusterRole to create secrets.  If the secret already exists
(either because an earlier instance of Ambassador Edge Stack already created
it, or because it was created manually), then the "create" permission
for secrets can be be removed from the ClusterRole.  If manually
providing the secret, it must have the "Opaque" type, with two data
fields: `rsa.key` and `rsa.crt`, which contain PEM-encoded RSA private
and public keys respectively.



[^1]: This may change in a future release to reflect the Pods's
    namespace if deployed to a namespace other than `default`.
    https://github.com/datawire/ambassador/issues/1583

[Go `net.Dial`]: https://golang.org/pkg/net/#Dial
[Go `strconv.ParseBool`]: https://golang.org/pkg/strconv/#ParseBool
