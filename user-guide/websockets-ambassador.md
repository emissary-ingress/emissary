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

