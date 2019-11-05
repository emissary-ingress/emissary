set -e

sudo cp /buildroot/bin/amb-sidecar /ambassador/sidecars
sudo touch /ambassador/.edge_stack

if [ ! -d /ambassador/init-config ]; then
	sudo mkdir /ambassador/init-config
fi

cat > /tmp/edge-stack-mappings.yaml <<EOF
---
apiVersion: getambassador.io/v1
kind: Mapping
metadata:
  name: edgestack-fallback-mapping
spec:
  prefix: /
  rewrite: /edge_stack_ui/
  service: 127.0.0.1:8500
  precedence: -1000000
---
apiVersion: getambassador.io/v1
kind: Mapping
metadata:
  name: edgestack-acme-mapping
spec:
  prefix: /.well-known/acme-challenge/
  rewrite: /.well-known/acme-challenge/
  service: 127.0.0.1:8500
EOF

sudo mv /tmp/edge-stack-mappings.yaml /ambassador/init-config
