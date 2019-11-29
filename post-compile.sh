set -e

sudo cp /buildroot/bin/amb-sidecar /ambassador/sidecars
sudo touch /ambassador/.edge_stack

sudo mkdir -p /ambassador/webui/bindata && sudo rsync -a --delete /buildroot/apro/cmd/amb-sidecar/webui/bindata/  /ambassador/webui/bindata

(
  cd /ambassador/webui/bindata/
  webpack --config webpack.config.js
)

sudo rm -rf /ambassador/init-config
sudo mkdir /ambassador/init-config

cat > /tmp/edge-stack-mappings.yaml <<EOF
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: edgestack-fallback-mapping
  labels:
    product: aes
    ambassador_diag_class: private
spec:
  prefix: /
  rewrite: /edge_stack_ui/
  service: 127.0.0.1:8500
  precedence: -1000000
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: edgestack-acme-mapping
  labels:
    product: aes
    ambassador_diag_class: private
spec:
  prefix: /.well-known/acme-challenge/
  rewrite: /.well-known/acme-challenge/
  service: 127.0.0.1:8500
  precedence: 1000000
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: ambassador-edge-stack
  namespace: ambassador
  labels:
    product: aes
    ambassador_diag_class: private
spec:
  prefix: /.ambassador/
  rewrite: ""
  service: "127.0.0.1:8500"  
  precedence: 1000000
EOF

sudo mv /tmp/edge-stack-mappings.yaml /ambassador/init-config

# Hack to have ambassador.version contain the apro.version info,
# because teaching VERSION.py to read apro.version seems like it will
# take too much work in the short term.
sudo cp -f /buildroot/ambassador.version /buildroot/ambassador/python/ambassador.version.bak
sudo cp -f /buildroot/ambassador/python/{apro,ambassador}.version
