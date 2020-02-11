set -e

eval "$(grep BUILD_VERSION /buildroot/apro.version 2>/dev/null)"
mkdir -p /buildroot/bin-darwin
(cd /buildroot/apro && GOOS=darwin go build -trimpath ${BUILD_VERSION:+ -ldflags "-X main.Version=$BUILD_VERSION" } -o /buildroot/bin-darwin ./cmd/aes-plugin-runner)

sudo cp /buildroot/bin/amb-sidecar /ambassador/sidecars
sudo cp /buildroot/bin/aes-plugin-runner /ambassador
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
