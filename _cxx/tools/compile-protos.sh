#!/bin/bash

BLUE='\033[0;34m'
GREEN='\033[0;32m'
NC='\033[0m'

OSS_SOURCE="${PWD}"

# envoy directories
BASE_ENVOY_DIR="${OSS_SOURCE}/_cxx/envoy"
ENVOY_PROTO_API_BASE="${BASE_ENVOY_DIR}/api"
ENVOY_COMPILED_GO_BASE="${BASE_ENVOY_DIR}/build_go"

# Emissary directories
EMISSARY_PROTO_API_BASE="${OSS_SOURCE}/api"
EMISSARY_COMPILED_PROTO_GO_BASE="${OSS_SOURCE}/pkg/api"

# envoy build container settings
ENVOY_DOCKER_OPTIONS="--platform=${BUILD_ARCH}"
export ENVOY_DOCKER_OPTIONS

# unset ssh auth sock because we don't need it in the container and
# the `run_envoy_docker.sh` adds it by default.
SSH_AUTH_SOCK=""
export SSH_AUTH_SOCK

############### copy raw protos into emissary repo ######################

echo -e "${BLUE}removing existing Envoy Protobuf API from:${GREEN} $EMISSARY_PROTO_API_BASE/envoy";
rm -rf "${EMISSARY_PROTO_API_BASE}/envoy"

echo -e "${BLUE}copying Envoy Protobuf API from ${GREEN} ${ENVOY_PROTO_API_BASE}/envoy ${NC}into ${GREEN}${EMISSARY_PROTO_API_BASE}/envoy";
rsync --recursive --delete --delete-excluded --prune-empty-dirs --include='*/' \
  --include='*.proto' --exclude='*' \
  "${ENVOY_PROTO_API_BASE}/envoy" "${EMISSARY_PROTO_API_BASE}"

echo -e "${BLUE}removing existing Envoy Contrib Protobuf API from:${GREEN} ${EMISSARY_PROTO_API_BASE}/contrib";
rm -rf "${EMISSARY_PROTO_API_BASE}/contrib"
mkdir -p "${EMISSARY_PROTO_API_BASE}/contrib/envoy/extensions/filters/http"

echo -e "${BLUE}copying Envoy Contrib Protobuf API from ${GREEN} ${ENVOY_PROTO_API_BASE}/contrib ${NC}into ${GREEN}${EMISSARY_PROTO_API_BASE}/contrib";
rsync --recursive --delete --delete-excluded --prune-empty-dirs \
  --include='*/' \
  --include='*.proto' \
  --exclude='*' \
  "${ENVOY_PROTO_API_BASE}/contrib/envoy/extensions/filters/http/golang" \
  "${ENVOY_PROTO_API_BASE}/contrib/envoy/extensions/filters/http/wasm" \
  "${ENVOY_PROTO_API_BASE}/contrib/envoy/extensions/filters/http/ext_proc" \
  "${EMISSARY_PROTO_API_BASE}/contrib/envoy/extensions/filters/http"

############### compile go protos ######################

echo -e "${BLUE}compiling go-protobufs in envoy build container${NC}";
rm -rf "${ENVOY_COMPILED_GO_BASE}"

cd "${BASE_ENVOY_DIR}" || exit;
./ci/run_envoy_docker.sh "./ci/do_ci.sh 'api.go'";
cd "${OSS_SOURCE}" || exit;

############## moving envoy compiled protos to emissary #################
echo -e "${BLUE}removing existing compiled protos from: ${GREEN} $EMISSARY_COMPILED_PROTO_GO_BASE/envoy${NC}";
rm -rf "${EMISSARY_COMPILED_PROTO_GO_BASE}/envoy"

echo -e "${BLUE}copying compiled protos from: ${GREEN} ${ENVOY_COMPILED_GO_BASE}/envoy${NC} into ${GREEN}${EMISSARY_COMPILED_PROTO_GO_BASE}/envoy${NC}";
rsync --recursive --delete --delete-excluded --prune-empty-dirs \
  --include='*/' \
  --include='*.go' \
  --exclude='*' \
  "${ENVOY_COMPILED_GO_BASE}/envoy" "${EMISSARY_COMPILED_PROTO_GO_BASE}"

echo -e "${BLUE}Updating import pkg references from: ${GREEN}github.com/envoyproxy/go-control-plane/envoy ${NC}--> ${GREEN}github.com/emissary-ingress/emissary/v3/pkg/api/envoy${NC}"
find "${EMISSARY_COMPILED_PROTO_GO_BASE}/envoy" -type f \
  -exec chmod 644 {} + \
  -exec sed -E -i.bak \
    -e 's,github\.com/envoyproxy/go-control-plane/envoy,github.com/emissary-ingress/emissary/v3/pkg/api/envoy,g' \
    -- {} +;

find "${EMISSARY_COMPILED_PROTO_GO_BASE}/envoy" -name '*.bak' -delete;

gofmt -w -s "${EMISSARY_COMPILED_PROTO_GO_BASE}/envoy"

############## moving contrib compiled protos to emissary #################
echo -e "${BLUE}removing existing compiled protos from: ${GREEN} $EMISSARY_COMPILED_PROTO_GO_BASE/contrib${NC}";
rm -rf "${EMISSARY_COMPILED_PROTO_GO_BASE}/contrib"
mkdir -p "${EMISSARY_COMPILED_PROTO_GO_BASE}/contrib/envoy/extensions/filters/http"

echo -e "${BLUE}copying compiled protos from: ${GREEN} ${ENVOY_COMPILED_GO_BASE}/contrib${NC} into ${GREEN}${EMISSARY_COMPILED_PROTO_GO_BASE}/contrib${NC}";
rsync --recursive --delete --delete-excluded --prune-empty-dirs \
  --include='*/' \
  --include='*.go' \
  --exclude='*' \
  "${ENVOY_COMPILED_GO_BASE}/contrib/envoy/extensions/filters/http/golang" \
  "${ENVOY_COMPILED_GO_BASE}/contrib/envoy/extensions/filters/http/wasm" \
  "${ENVOY_COMPILED_GO_BASE}/contrib/envoy/extensions/filters/http/ext_proc" \
  "${EMISSARY_COMPILED_PROTO_GO_BASE}/contrib/envoy/extensions/filters/http"

echo -e "${BLUE}Updating import pkg references from: ${GREEN}github.com/envoyproxy/go-control-plane/envoy ${NC}--> ${GREEN}github.com/emissary-ingress/emissary/v3/pkg/api/envoy${NC}"
find "${EMISSARY_COMPILED_PROTO_GO_BASE}/contrib" -type f \
  -exec chmod 644 {} + \
  -exec sed -E -i.bak \
    -e 's,github\.com/envoyproxy/go-control-plane/envoy,github.com/emissary-ingress/emissary/v3/pkg/api/envoy,g' \
    -- {} +;

find "${EMISSARY_COMPILED_PROTO_GO_BASE}/contrib" -name '*.bak' -delete;

gofmt -w -s "${EMISSARY_COMPILED_PROTO_GO_BASE}/contrib"
