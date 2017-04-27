HERE=$(dirname $0)
eval $(sh $HERE/../scripts/get_registries.sh)

if [ -z "${DOCKER_REGISTRY}" ]; then exit 1; fi

cat <<EOF
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  creationTimestamp: null
  name: statsd-sink
spec:
  replicas: 1
  strategy: {}
  template:
    metadata:
      creationTimestamp: null
      labels:
        service: statsd-sink
    spec:
      containers:
      - name: statsd-sink
        image: ${STREG}prom-statsd-exporter:0.5.2
        resources: {}
      restartPolicy: Always
status: {}
---
apiVersion: v1
kind: Service
metadata:
  creationTimestamp: null
  labels:
    service: statsd-sink
  name: statsd-sink
spec:
  ports:
  - protocol: UDP
    port: 8125
    name: statsd-metrics
  - protocol: TCP
    port: 9102
    name: prometheus-metrics
  selector:
    service: statsd-sink
EOF
