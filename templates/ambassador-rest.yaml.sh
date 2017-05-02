HERE=$(dirname $0)
eval $(sh $HERE/../scripts/get_registries.sh)

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
        image: ${AMREG}ambassador:0.7.0
        # ports:
        # - containerPort: 80
        #   protocol: TCP
        resources:
          limits:
            cpu: 1
            memory: 400Mi
          requests:
            cpu: 200m
            memory: 100Mi
        volumeMounts:
        - mountPath: /etc/certs
          name: cert-data
      - name: statsd
        image: ${STREG}statsd:0.7.0
        resources: {}
      volumes:
      - name: cert-data
        secret:
          secretName: ambassador-certs
      restartPolicy: Always
status: {}
EOF
