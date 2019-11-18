# Headers

Ambassador Edge Stack can route to target services based on HTTP headers with the `headers` and `regex_headers` annotations. Multiple mappings with different annotations can be applied to construct more complex routing rules.

## The `headers` annotation

The `headers` attribute is a dictionary of `header`: `value` pairs. Ambassador Edge Stack will only allow requests that match the specified `header`: `value` pairs to reach the target service.

You can also set the `value` of a header to `true` to test for the existence of a header.
