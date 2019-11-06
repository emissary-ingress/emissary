# Method-based routing

Ambassador Edge Stack supports routing based on HTTP method and regular expression.

## Using `method`

The `method` annotation specifies the specific HTTP method for a mapping. The value of the `method` annotation must be in all upper case.

For example:

```yaml
---
aapiVersion: ambassador/v1
kind: Mapping
name: get_mapping
prefix: /backend/get_only/
method: GET
service: tour
```

## Using `method_regex`

When `method_regex` is set to `true`, the value of the `method` annotation will be interpreted as a regular expression. 

<div style="border: solid gray;padding:0.5em">

Ambassador Edge Stack is a community supported product with [features](getambassador.io/features) available for free and limited use. For unlimited access and commercial use of Ambassador Edge Stack, [contact sales](https:/www.getambassador.io/contact) for access to [Ambassador Edge Stack Enterprise](/user-guide/ambassador-edge-stack-enterprise) today.

</div>
