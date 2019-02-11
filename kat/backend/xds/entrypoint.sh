#!/usr/bin/env bash
set -o errexit
set -o nounset
set -o pipefail

# This script was adapted from https://github.com/envoyproxy/go-control-plane, so that
# when can build from any custom repo and not only envoy upstream. It's possible
# that some of these dependencies will be out of date or even deprecated when updating the
# project. In this case, check if the dependencies bellow and in the glide.yaml match the
# the dependencies from the Envoy api SHA being used.

package_path="github.com/datawire/ambassador/kat/backend/xds/"
root="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
xds=${root}/proto

echo "Expecting protoc version >= 3.5.0:"
${root}/bin/protoc --version

imports=(
  ${xds}
  "${root}/vendor/github.com/lyft/protoc-gen-validate"
  "${root}/vendor/github.com/gogo/protobuf"
  "${root}/vendor/github.com/gogo/protobuf/protobuf"
  "${root}/vendor/istio.io/gogo-genproto/prometheus"
  "${root}/vendor/istio.io/gogo-genproto/googleapis"
  "${root}/vendor/istio.io/gogo-genproto/opencensus/proto/trace/v1"
)

protocarg=""
for i in "${imports[@]}"
do
  protocarg+="--proto_path=$i "
done

mappings=(
  "google/api/annotations.proto=github.com/gogo/googleapis/google/api"
  "google/api/http.proto=github.com/gogo/googleapis/google/api"
  "google/rpc/code.proto=github.com/gogo/googleapis/google/rpc"
  "google/rpc/error_details.proto=github.com/gogo/googleapis/google/rpc"
  "google/rpc/status.proto=github.com/gogo/googleapis/google/rpc"
  "google/protobuf/any.proto=github.com/gogo/protobuf/types"
  "google/protobuf/duration.proto=github.com/gogo/protobuf/types"
  "google/protobuf/empty.proto=github.com/gogo/protobuf/types"
  "google/protobuf/struct.proto=github.com/gogo/protobuf/types"
  "google/protobuf/timestamp.proto=github.com/gogo/protobuf/types"
  "google/protobuf/wrappers.proto=github.com/gogo/protobuf/types"
  "gogoproto/gogo.proto=github.com/gogo/protobuf/gogoproto"
  "trace.proto=istio.io/gogo-genproto/opencensus/proto/trace/v1"
  "metrics.proto=istio.io/gogo-genproto/prometheus"
)

gogoarg="plugins=grpc"

for mapping in "${mappings[@]}"
do
  gogoarg+=",M$mapping"
done

for path in $(find ${xds} -type d)
do  
  path_protos=(${path}/*.proto)
  if [[ ${#path_protos[@]} > 0 ]]
  then
    for path_proto in "${path_protos[@]}"
    do  
      if [ -f "$path_proto" ]; then
        mapping=${path_proto##${xds}/}=${package_path}${path##${xds}/}
        gogoarg+=",M$mapping"
      fi
    done
  fi
done

for path in $(find ${xds} -type d)
do
  path_protos=(${path}/*.proto)
  if [[ ${#path_protos[@]} > 0 ]]
  then
    for path_proto in "${path_protos[@]}"
    do
      if [ -f "$path_proto" ]; then  
        echo "Generating go file ${path} ..."
        ${root}/bin/protoc ${protocarg} ${path}/*.proto \
          --plugin=protoc-gen-gogofast=${root}/bin/gogofast --gogofast_out=${gogoarg}:. \
          --plugin=protoc-gen-validate=${root}/bin/validate --validate_out="lang=gogo:."
      fi
    done
  fi
done

cp -r ${root}/envoy/* /envoy
cp -r ${root}/vendor/* /vendor
