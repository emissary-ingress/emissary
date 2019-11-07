# Prefix regex

## Using `prefix` and `prefix_regex`

When the `prefix_regex` attribute is set to `true`, Ambassador Edge Stack configures a [regex route](https://www.envoyproxy.io/docs/envoy/v1.5.0/api-v1/route_config/route#route) instead of a prefix route in envoy. **This means the entire path must match the regex specified, not only the prefix.**

## Example with version in url

If the version is a path parameter and the resources are served by different services, then

```yaml
---
apiVersion: getambassador.io/v2
kind:  Mapping
name:  qotm_mapping
prefix: "/(v1|v2)/qotm/.*"
prefix_regex: true
service: qotm
```

will map requests to both `/v1` and `/v2` to the `qotm` service.

Note that enclosing regular expressions in quotes can be important to prevent backslashes from being doubled.

<div style="border: thick solid gray;padding:0.5em"> 

Ambassador Edge Stack is a community supported product with 
[features](getambassador.io/features) available for free and 
limited use. For unlimited access and commercial use of
Ambassador Edge Stack, [contact sales](https:/www.getambassador.io/contact) 
for access to [Ambassador Edge Stack Enterprise](/user-guide/ambassador-edge-stack-enterprise) today.

</div>
</p>
