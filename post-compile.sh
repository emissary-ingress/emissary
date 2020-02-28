#!/hint/bash
set -e

eval "$(grep BUILD_VERSION /buildroot/apro.version 2>/dev/null)"
mkdir -p /buildroot/bin-darwin
(cd /buildroot/apro && GOOS=darwin go build -trimpath ${BUILD_VERSION:+ -ldflags "-X main.Version=$BUILD_VERSION" } -o /buildroot/bin-darwin ./cmd/aes-plugin-runner)

# Create symlinks to the multi-call binary so the original names can be used in
# the builder shell easily (from the shell PATH).
ln -sf /buildroot/bin/ambassador /buildroot/bin/ambex
ln -sf /buildroot/bin/ambassador /buildroot/bin/kubestatus
ln -sf /buildroot/bin/ambassador /buildroot/bin/watt
ln -sf /buildroot/bin/ambassador /buildroot/bin/amb-sidecar
ln -sf /buildroot/bin/ambassador /buildroot/bin/app-sidecar
ln -sf /buildroot/bin/ambassador /buildroot/bin/aes-plugin-runner

# Also note there is a different ambassador binary, written in Python, that
# shows up earlier in the shell PATH:
#   $ type -a ambassador
#   ambassador is /usr/bin/ambassador
#   ambassador is /buildroot/bin/ambassador

# Stuff in /opt/ambassador/bin in the builder winds up in /usr/local/bin in the
# production image.
sudo install -D -t /opt/ambassador/bin/ /buildroot/bin/ambassador
sudo ln -sf /opt/ambassador/bin/ambassador /opt/ambassador/bin/ambex
sudo ln -sf /opt/ambassador/bin/ambassador /opt/ambassador/bin/kubestatus
sudo ln -sf /opt/ambassador/bin/ambassador /opt/ambassador/bin/watt
sudo ln -sf /opt/ambassador/bin/ambassador /opt/ambassador/bin/amb-sidecar
sudo ln -sf /opt/ambassador/bin/ambassador /opt/ambassador/bin/app-sidecar
sudo ln -sf /opt/ambassador/bin/ambassador /opt/ambassador/bin/aes-plugin-runner

# Set things up for the plugin runner and for computing the ABI info
sudo ln -sf /opt/ambassador/bin/amb-sidecar /ambassador/sidecars/
sudo ln -sf /opt/ambassador/bin/aes-plugin-runner /ambassador/

sudo touch /ambassador/.edge_stack

sudo mkdir -p /ambassador/webui/bindata && sudo make -f build-aux-local/minify.mk

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
#
# 2020-01-30: Removing this hack speeds up builds.  Since they're
# released in lockstep, it shouldn't matter anymore?
#sudo cp -f /buildroot/ambassador.version /buildroot/ambassador/python/ambassador.version.bak
#sudo cp -f /buildroot/ambassador/python/{apro,ambassador}.version

{
  echo "# _GOVERSION=$(go version /ambassador/sidecars/amb-sidecar | sed 's/.*go//')"
  echo "# GOPATH=$(go env GOPATH)"
  echo '# GOOS=linux'
  echo '# GOARCH=amd64'
  echo '# CGO_ENABLED=1'
  echo '# GO111MODULE=on'
  go version -m /ambassador/sidecars/amb-sidecar | awk '$1 == "dep" && $4 ~ /^h1:/ { print $2, $3 }'
} > /tmp/aes-abi.txt
sudo mv /tmp/aes-abi.txt /ambassador/
