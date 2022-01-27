httpbin_manifests="""
---
apiVersion: v1
kind: Service
metadata:
  name: httpbin
spec:
  type: ClusterIP
  selector:
    service: httpbin
  ports:
  - port: 80
    targetPort: http
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: httpbin
spec:
  replicas: 1
  selector:
    matchLabels:
      service: httpbin
  template:
    metadata:
      labels:
        service: httpbin
    spec:
      containers:
      - name: httpbin
        image: kennethreitz/httpbin
        ports:
        - name: http
          containerPort: 80
"""

qotm_manifests = """
---
apiVersion: v1
kind: Service
metadata:
  name: qotm
spec:
  selector:
    service: qotm
  ports:
    - port: 80
      targetPort: http-api
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: qotm
spec:
  selector:
    matchLabels:
      service: qotm
  replicas: 1
  strategy:
    type: RollingUpdate
  template:
    metadata:
      annotations:
        sidecar.istio.io/inject: "false"
      labels:
        service: qotm
    spec:
      serviceAccountName: ambassador
      containers:
      - name: qotm
        image: docker.io/datawire/qotm:1.3
        ports:
        - name: http-api
          containerPort: 5000
"""


websocket_echo_server_manifests="""
---
apiVersion: v1
kind: Service
metadata:
  name: websocket-echo-server
spec:
  selector:
    service: websocket-echo-server
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: websocket-echo-server
spec:
  selector:
    matchLabels:
      service: websocket-echo-server
  replicas: 1
  strategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        service: websocket-echo-server
    spec:
      containers:
      - name: websocket-echo-server
        image: docker.io/johnesmet/go-websocket-echo-server:latest
"""


# This is a little weird -- you need to fill in the '%s' with the namespace
# you want before you use 'format' to fill in other things from 'self'.
cleartext_host_manifest = """
---
apiVersion: getambassador.io/v3alpha1
kind: Host
metadata:
  name: cleartext-host-{self.path.k8s}
  labels:
    scope: AmbassadorTest
  namespace: %s
spec:
  ambassador_id: [ "{self.ambassador_id}" ]
  hostname: "*"
  acmeProvider:
    authority: none
  requestPolicy:
    insecure:
      action: Route
"""

# This is a little weird -- there are several things to fill in before you
# use 'format' to fill in other things from 'self'.
default_listener_manifest = """
---
apiVersion: getambassador.io/v3alpha1
kind: Listener
metadata:
  name: listener-%(port)s-{self.path.k8s}
  labels:
    scope: AmbassadorTest
  namespace: %(namespace)s
spec:
  ambassador_id: [ "{self.ambassador_id}" ]
  port: %(port)d
  protocol: %(protocol)s
  securityModel: %(securityModel)s
  hostBinding:
    namespace:
      from: ALL
"""
