# WebSockets and Ambassador Edge Stack

Ambassador Edge Stack makes it easy to access your services from outside your application, and this includes services that use WebSockets. Only a small amount of additional configuration is required, which is as simple as adding the `use_websocket` attribute with a value of `true` on a `Mapping`.

## Writing a WebSocket service for Ambassador Edge Stack

The example configuration below demonstrates the addition of the `use_websocket` attribute.

```yaml
kind: Service
apiVersion: v1
metadata:
  name: my-service
  annotations:
    getambassador.io/config: |
      ---
        apiVersion: getambassador.io/v2
        kind:  Mapping
        name:  my_service_mapping
        prefix: /my-service/
        service: my-service
        use_websocket: true
spec:
  selector:
    app: MyApp
  ports:
  - protocol: TCP
    port: 80
    targetPort: 9376
```

<div style="border: thick solid gray;padding:0.5em"> 

Ambassador Edge Stack is a community supported product with 
[features](getambassador.io/features) available for free and 
limited use. For unlimited access and commercial use of
Ambassador Edge Stack, [contact sales](https:/www.getambassador.io/contact) 
for access to [Ambassador Edge Stack Enterprise](/user-guide/ambassador-edge-stack-enterprise) today.

</div>
</p>