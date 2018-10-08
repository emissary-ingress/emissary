## Header-based routing with regular expressions

The `regex_headers` annotation specifies a list of HTTP headers which must match in order for the mapping to be used.

### Example

The following mapping will route mobile requests from Android and iPhones to a mobile service:

```yaml
name: mobile-ui
  annotations:
    getambassador.io/config: |
      ---
      apiVersion: ambassador/v0
      kind:  Mapping
      name:  mobile_ui_mapping
      regex_headers:
        user-agent: "^(?=.*\\bAndroid\\b)(?=.*\\b(m|M)obile\\b).*|(?=.*\\biPhone\\b)(?=.*\\b(m|M)obile\\b).*$"
      prefix: /
      service: mobile-ui
```