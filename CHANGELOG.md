## Ambassador Pro CHANGELOG


## 0.4.0

Moved all of the default sidecar ports around; YAML will need to be adjusted (hence 0.4.0 instead of 0.3.2).  Additionally, all of the ports are now configurable via environment variables.

        ```
        | purpose          | env-var        |  old |  new |
        |------------------+----------------+------+------|
        | Auth gRPC        | APRO_AUTH_PORT | 8082 | 8500 |
        | RLS gRPC         | GRPC_PORT      | 8081 | 8501 |
        | RLS debug (HTTP) | DEBUG_PORT     | 6070 | 8502 |
        | RLS HTTP ???     | PORT           | 7000 | 8503 |
        ```

## 0.3.1

??

## 0.3.0

* Filter type `External`
* Request IDs in the Pro logs are the same as the Request IDs in the Ambassador logs
* `Oauth2` filter type supports `secretName` and `secretNamespace`
* Switch to using Ambassador OSS gRPC API
* RLS logs requests as `info` instead of `warn`
