# Gzip Compression

Gzip enables Ambassador Edge Stack to compress upstream data upon client request. Compression is useful in situations where large payloads need to be transmitted without compromising the response time. Compression can also save on bandwidth costs at the expense of increased computing costs.

## How Does it Work?

When the gzip filter is enabled, request and response headers are inspected to determine whether or not the content should be compressed. If so, and the request and response headers allow, the content is compressed and then sent to the client with the appropriate headers. It also uses the zlib module, which provides `Deflate` compression and decompression code.

For more details see [Envoy - Gzip](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/gzip_filter.html).

## The `gzip` API

- `memory_level`: A value from 1 to 9 that controls the amount of internal memory used by zlib. Higher values use more memory, but are faster and produce better compression results. The default value is 5.
- `min_content_length`: A minimum response length, in bytes, which will trigger compression. The default value is 30.
- `compression_level`: A value used for selecting the zlib compression level. This setting will affect the speed and amount of compression applied to the content. “BEST” provides higher compression at the cost of higher latency, “SPEED” provides lower compression with minimum impact on response time. “DEFAULT” provides an optimal result between speed and compression. This field will be set to “DEFAULT” if not specified.
- `compression_strategy`: A value used for selecting the zlib compression strategy which is directly related to the characteristics of the content. Most of the time “DEFAULT” will be the best choice, though there are situations in which changing this parameter might produce better results. For example, run-length encoding (RLE) is typically used when the content is known for having sequences in which the same data occurs many consecutive times. For more information about each strategy, please refer to the zlib manual.
- `window_bits`: A value from 9 to 15 that represents the base two logarithmic of the compressor’s window size. Larger window results in better compression at the expense of memory usage. The default is 12 which will produce a 4096 bytes window. For more details about this parameter, please refer to zlib manual > deflateInit2.
- `content_type`: A set of strings that specify which mime-types yield compression; e.g., application/json, text/html, etc. When this field is not defined, compression will be applied to the following mime-types: “application/javascript”, “application/json”, “application/xhtml+xml”, “image/svg+xml”, “text/css”, “text/html”, “text/plain”, “text/xml”.
- `disable_on_etag_header`: A Boolean, if true, disables compression when the response contains an `etag` header. When it is false, the filter will preserve weak `etag`s and remove the ones that require strong validation.
- `remove_accept_encoding_header`: A Boolean, if true, removes accept-encoding from the request headers before dispatching it to the upstream so that responses do not get compressed before reaching the filter.

## Example

```yaml
apiVersion: getambassador.io/v2
kind:  Module
metadata:
  name:  ambassador
spec:
  config:
    gzip:
      memory_level: 2
      min_content_length: 32
      compression_level: BEST
      compression_strategy: RLE
      content_type: 
      - application/javascript
      - application/json
      - text/plain
      disable_on_etag_header: false
      remove_accept_encoding_header: false
```

Minimum configuration:

```yaml
apiVersion: getambassador.io/v2
kind:  Module
metadata:
  name:  ambassador
spec:
  config:
    gzip:
      enabled: true
```
