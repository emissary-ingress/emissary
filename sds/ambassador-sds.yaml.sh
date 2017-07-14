HERE=$(dirname $0)
eval $(sh $HERE/../scripts/get_registries.sh)

if [ -z "${DOCKER_REGISTRY}" ]; then exit 1; fi

cat <<EOF
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  creationTimestamp: null
  name: ambassador-sds
spec:
  replicas: 1
  strategy: {}
  template:
    metadata:
      creationTimestamp: null
      labels:
        service: ambassador-sds
    spec:
      containers:
      - name: ambassador-sds
        image: {{AMREG}}ambassador-sds:0.6.0
        resources: {}
      restartPolicy: Always
status: {}
---
apiVersion: v1
kind: Service
metadata:
  creationTimestamp: null
  labels:
    service: ambassador-sds
  name: ambassador-sds
spec:
  type: NodePort
  ports:
  - name: ambassador-sds
    port: 5000
    targetPort: 5000
  selector:
    service: ambassador-sds
EOF
