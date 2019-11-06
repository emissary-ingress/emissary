# Remove request headers

Ambassador Edge Stack can remove a list of HTTP headers that would be sent to the upstream from the request.

## The `remove_request_headers` annotation

The `remove_request_headers` attribute takes a list of keys used to match to the header

## A basic example

```yaml
---
apiVersion: ambassador/v1
kind:  Mapping
name:  tour-ui_mapping
prefix: /
remove_request_headers:
- authorization
service: tour
```

will drop header with key `authorization`.

<div style="border: solid gray;padding:0.5em">

Ambassador Edge Stack is a community supported product with [features](getambassador.io/features) available for free and limited use. For unlimited access and commercial use of Ambassador Edge Stack, [contact sales](https:/www.getambassador.io/contact) for access to [Ambassador Edge Stack Enterprise](/user-guide/ambassador-edge-stack-enterprise) today.

</div>
