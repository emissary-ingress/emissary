# Using WebSockets and Ambassador

Ambassador Edge Stack makes it easy to access your services from outside your application, and this includes services that use WebSockets, such as [Github](https://github.com/websockets). Only a small amount of additional configuration is required, which is as simple as adding the `use_websocket` attribute with a value of `true` on a `Mapping`.

## Example Websocket Service

The example configuration below demonstrates the addition of the `use_websocket` attribute.

```yaml
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: my-service-mapping
spec:
  prefix: /my-service/
  service: my-service
  use_websocket: true

---
kind: Service
apiVersion: v1
metadata:
  name: my-service
spec:
  selector:
    app: MyApp
  ports:
  - protocol: TCP
    port: 80
    targetPort: 9376
```
