set -e

sudo cp /buildroot/bin/amb-sidecar /ambassador/sidecars
sudo touch /ambassador/.edge_stack

sudo mkdir -p /ambassador/webui/bindata && sudo rsync -a --delete /buildroot/apro/cmd/amb-sidecar/webui/bindata/  /ambassador/webui/bindata
(
  cd /ambassador/webui/bindata
  # At this time we don't want to generate a bundle. Use rollup only to minify each individual files.
  for file in $PWD/edge_stack/components/*.js
  do
    NODE_PATH="$(npm root -g)" rollup -c rollup.config.js -i $file -o $file
  done
)

sudo rm -rf /ambassador/init-config
sudo mkdir /ambassador/init-config

cat > /tmp/edge-stack-mappings.yaml <<EOF
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: edgestack-fallback-mapping
  namespace: _automatic_
  labels:
    product: aes
    ambassador_diag_class: private
spec:
  ambassador_id: [ "_automatic_" ]
  prefix: /
  rewrite: /edge_stack_ui/
  service: 127.0.0.1:8500
  precedence: -1000000
  timeout_ms: 60000
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: edgestack-direct-mapping
  namespace: _automatic_
  labels:
    product: aes
    ambassador_diag_class: private
spec:
  ambassador_id: [ "_automatic_" ]
  prefix: /edge_stack/
  rewrite: /edge_stack_ui/edge_stack/
  service: 127.0.0.1:8500
  precedence: 1000000
  timeout_ms: 60000
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: ambassador-edge-stack
  namespace: _automatic_
  labels:
    product: aes
    ambassador_diag_class: private
spec:
  ambassador_id: [ "_automatic_" ]
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
