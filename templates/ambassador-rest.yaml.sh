HERE=$(dirname $0)
eval $(sh $HERE/../scripts/get_registries.sh)

if [ -z "${DOCKER_REGISTRY}" ]; then exit 1; fi

cat <<EOF
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  creationTimestamp: null
  name: ambassador
spec:
  replicas: 1
  strategy: {}
  template:
    metadata:
      creationTimestamp: null
      labels:
        service: ambassador
        # service: ambassador-admin
    spec:
      containers:
      - name: ambassador
        image: ${AMREG}ambassador:0.5.1
        # ports:
        # - containerPort: 80
        #   protocol: TCP
        resources: {}
        volumeMounts:
        - mountPath: /etc/certs
          name: cert-data
      - name: statsd
        image: ${STREG}statsd:0.5.1
        resources: {}
      volumes:
      - name: cert-data
        secret:
          secretName: ambassador-certs
      restartPolicy: Always
status: {}
EOF
