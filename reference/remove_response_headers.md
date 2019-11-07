# Remove response headers

Ambassador Edge Stack can remove a list of HTTP headers that would be sent to the client in the response (eg. default `x-envoy-upstream-service-time`)

## The `remove_response_headers` annotation

The `remove_response_headers` attribute takes a list of keys used to match to the header

## A basic example

```yaml
---
apiVersion: getambassador.io/v2
kind:  Mapping
name:  tour-ui_mapping
prefix: /
remove_response_headers:
- x-envoy-upstream-service-time
service: tour
```

will drop header with key `x-envoy-upstream-service-time`.

<div style="border: thick solid gray;padding:0.5em"> 

Ambassador Edge Stack is a community supported product with 
[features](getambassador.io/features) available for free and 
limited use. For unlimited access and commercial use of
Ambassador Edge Stack, [contact sales](https:/www.getambassador.io/contact) 
for access to [Ambassador Edge Stack Enterprise](/user-guide/ambassador-edge-stack-enterprise) today.

</div>
</p>
