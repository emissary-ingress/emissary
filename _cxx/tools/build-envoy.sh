#!/bin/bash

# The phony make targets have been exported when calling from Make.
FIPS_MODE=${FIPS_MODE:-}
BUILD_ARCH=${BUILD_ARCH:-linux/amd64}

# base directory vars
OSS_SOURCE="$PWD"
BASE_ENVOY_DIR="$PWD/_cxx/envoy"
ENVOY_DOCKER_BUILD_DIR="$PWD/_cxx/envoy-docker-build"
export ENVOY_DOCKER_BUILD_DIR

# container vars
DOCKER_OPTIONS=(
  "--platform=${BUILD_ARCH}"
  "--env=ENVOY_DELIVERY_DIR=/build/envoy/x64/contrib/exe/envoy"
  "--env=ENVOY_BUILD_TARGET=//contrib/exe:envoy-static"
  "--env=ENVOY_BUILD_DEBUG_INFORMATION=//contrib/exe:envoy-static.dwp"
  # "--env=BAZEL_BUILD_OPTIONS=\-\-define tcmalloc=gperftools"
  )

# unset ssh auth sock because we don't need it in the container and
# the `run_envoy_docker.sh` adds it by default. This causes issues
# if trying to run builds on docker for mac.
SSH_AUTH_SOCK=""
export SSH_AUTH_SOCK

BAZEL_BUILD_EXTRA_OPTIONS=()
if [ -n "$FIPS_MODE" ]; then
  BAZEL_BUILD_EXTRA_OPTIONS+=(--define boringssl=fips)
fi;

if [ ! -d "$BASE_ENVOY_DIR" ]; then
  echo "Looks like Envoy hasn't been cloned locally yet, run clone-envoy target to ensure it is cloned";
  exit 1;
fi;

ENVOY_DOCKER_OPTIONS="${DOCKER_OPTIONS[*]}"
export ENVOY_DOCKER_OPTIONS

echo "Building custom build of Envoy using the following parameters:"
echo "   FIPS_MODE: ${FIPS_MODE}"
echo "   BUILD_ARCH: ${BUILD_ARCH}"
echo "   ENVOY_DOCKER_BUILD_DIR: ${ENVOY_DOCKER_BUILD_DIR}"
echo "   ENVOY_DOCKER_OPTIONS: ${ENVOY_DOCKER_OPTIONS}"
echo "   SSH_AUTH_SOCK: ${SSH_AUTH_SOCK}"
echo " "

ci_cmd="./ci/do_ci.sh 'release.server_only'"

if [ ${#BAZEL_BUILD_EXTRA_OPTIONS[@]} -gt 0  ]; then
  ci_cmd="BAZEL_BUILD_EXTRA_OPTIONS='${BAZEL_BUILD_EXTRA_OPTIONS[*]}' $ci_cmd"
fi;

echo "cleaning up any old build binaries"
rm -rf "$ENVOY_DOCKER_BUILD_DIR/envoy";

# build envoy
cd "${BASE_ENVOY_DIR}" || exit
./ci/run_envoy_docker.sh "${ci_cmd}"
cd "${OSS_SOURCE}" || exit

echo "Untar release distribution which includes static builds"
tar -xvf "${ENVOY_DOCKER_BUILD_DIR}/envoy/x64/bin/release.tar.zst" -C "${ENVOY_DOCKER_BUILD_DIR}/envoy/x64/bin";

echo "Copying envoy-static and envoy-static-stripped to 'docker/envoy-build'";
cp "${ENVOY_DOCKER_BUILD_DIR}/envoy/x64/bin/dbg/envoy-contrib" "${PWD}/docker/base-envoy/envoy-static"
chmod +x "${PWD}/docker/base-envoy/envoy-static"

cp "${ENVOY_DOCKER_BUILD_DIR}/envoy/x64/bin/dbg/envoy-contrib.dwp" "${PWD}/docker/base-envoy/envoy-static.dwp"
chmod +x "${PWD}/docker/base-envoy/envoy-static.dwp"

cp "${ENVOY_DOCKER_BUILD_DIR}/envoy/x64/bin/envoy-contrib" "${PWD}/docker/base-envoy/envoy-static-stripped"
chmod +x "${PWD}/docker/base-envoy/envoy-static-stripped"
