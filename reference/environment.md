# Environment variables for the Ambassador container

Use the following variables for the environment of your Ambassdor container:

| Purpose                                 | Variable                         | Default value                                     | Value type                                                                    |
|-----------------------------------------|----------------------------------|---------------------------------------------------|-------------------------------------------------------------------------------|
| Ambassador                              | `AMBASSADOR_ID`                  | `default`                                         | Plain string                                                                  |
| Ambassador                              | `AMBASSADOR_NAMESPACE`           | `default` ([^1])                                  | Kubernetes namespace                                                          |
| Ambassador                              | `AMBASSADOR_SINGLE_NAMESPACE`    | Empty                                             | Boolean; non-empty=true, empty=false                                          |
| Ambassador Edge Stack                   | `AES_LOG_LEVEL`                  | `info`                                            | Log level                                                                     |
| Ambassador Edge Stack                   | `REDIS_POOL_SIZE`                | `10`                                              | Integer                                                                       |
| Ambassador Edge Stack                   | `REDIS_SOCKET_TYPE`              | None, must be set manually                        | Go network such as `tcp` or `unix`; see [Go `net.Dial`][]                     |
| Ambassador Edge Stack                   | `REDIS_URL`                      | None, must be set manually                        | Go network address; for TCP this is a `host:port` pair; see [Go `net.Dial`][] |
| Ambassador Edge Stack: RateLimit        | `REDIS_PERSECOND`                | `false`                                           | Boolean; [Go `strconv.ParseBool`][]                                           |
| Ambassador Edge Stack: RateLimit        | `REDIS_PERSECOND_SOCKET_TYPE`    | None, must be set manually (if `REDIS_PERSECOND`) | Go network such as `tcp` or `unix`; see [Go `net.Dial`][]                     |
| Ambassador Edge Stack: RateLimit        | `REDIS_PERSECOND_POOL_SIZE`      | None, must be set manually (if `REDIS_PERSECOND`) | Go network address; for TCP this is a `host:port` pair; see [Go `net.Dial`][] |
| Ambassador Edge Stack: RateLimit        | `EXPIRATION_JITTER_MAX_SECONDS`  | `300`                                             | Integer                                                                       |
| Ambassador Edge Stack: RateLimit        | `USE_STATSD`                     | `false`                                           | Boolean; [Go `strconv.ParseBool`][]                                           |
| Ambassador Edge Stack: RateLimit        | `STATSD_HOST`                    | `localhost`                                       | Hostname                                                                      |
| Ambassador Edge Stack: RateLimit        | `STATSD_PORT`                    | `8125`                                            | Integer                                                                       |
| Ambassador Edge Stack: RateLimit        | `GOSTATS_FLUSH_INTERVAL_SECONDS` | `5`                                               | Integer                                                                       |
| Ambassador Edge Stack: Developer Portal | `AMBASSADOR_URL`                 | `https://api.example.com`                         | URL                                                                           |
| Ambassador Edge Stack: Developer Portal | `DEVPORTAL_CONTENT_URL`          | `https://github.com/datawire/devportal-content`   | git-remote URL                                                                |
| Ambassador Edge Stack: Developer Portal | `DEVPORTAL_CONTENT_DIR`          | `/`                                               | Rooted Git directory                                                          |
| Ambassador Edge Stack: Developer Portal | `DEVPORTAL_CONTENT_BRANCH`       | `master`                                          | Git branch name                                                               |
| Ambassador Edge Stack: Developer Portal | `POLL_EVERY_SECS`                | `60`                                              | Integer                                                                       |

## Other Considerations

**Port names:** well-known port names that are recognized by `/etc/services`, but they are ***not* Kubernetes port names.**

**Log levels:** case-insensitive. From least verbose to most verbose, valid log levels are `error`, `warn`/`warning`, `info`, `debug`, and `trace`.

**`REDIS`:**

* The AuthService and the RateLimitService share a Redis connection pool; there will be up to `REDIS_POOL_SIZE` connections to Redis.
* If `REDIS_PERSECOND` is true, a second Redis connection pool is created (to a potentially different Redis instance) that is only used for per-second RateLimits.


[^1]: This may change in a future release to reflect the Pods's
    namespace if deployed to a namespace other than `default`.
    https://github.com/datawire/ambassador/issues/1583

[Go `net.Dial`]: https://golang.org/pkg/net/#Dial
[Go `strconv.ParseBool`]: https://golang.org/pkg/strconv/#ParseBool
