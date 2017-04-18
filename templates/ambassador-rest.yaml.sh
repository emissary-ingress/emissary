if [ -z "${DOCKER_REGISTRY}" ]; then
    AMREG=dwflynn
    STREG=ark3
else
    AMREG="${DOCKER_REGISTRY}"
    STREG="${DOCKER_REGISTRY}"
fi
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
        image: ${AMREG}/ambassador:0.5.0
        # ports:
        # - containerPort: 80
        #   protocol: TCP
        resources: {}
        volumeMounts:
        - mountPath: /etc/certs
          name: cert-data
      - name: statsd
        image: ${STREG}/statsd:0.5.0
        resources: {}
      volumes:
      - name: cert-data
        secret:
          secretName: ambassador-certs
      restartPolicy: Always
status: {}
EOF
