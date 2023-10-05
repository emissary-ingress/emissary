#!/bin/bash

# Input Args capture from Environement Variables
# The phone make targets have been configured to pass these along when using Make.
default_test_targets="//contrib/golang/... //test/..."
FIPS_MODE=${FIPS_MODE:-}
BUILD_ARCH=${BUILD_ARCH:-linux/amd64}
ENVOY_TEST_LABEL=${ENVOY_TEST_LABEL:-$default_test_targets}

# static vars
OSS_SOURCE="$PWD"
BASE_ENVOY_DIR="$PWD/_cxx/envoy"
ENVOY_DOCKER_BUILD_DIR="$PWD/_cxx/envoy-docker-build"
export ENVOY_DOCKER_BUILD_DIR

# Dynamic variables
DOCKER_OPTIONS=(
  "--platform=${BUILD_ARCH}"
  "--network=host"
  )

ENVOY_DOCKER_OPTIONS="${DOCKER_OPTIONS[*]}"
export ENVOY_DOCKER_OPTIONS

# unset ssh auth sock because we don't need it in the container and
# the `run_envoy_docker.sh` adds it by default.
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


echo "Running Envoy Tests with the following parameters set:"
echo "   ENVOY_TEST_LABEL: ${ENVOY_TEST_LABEL}"
echo "   FIPS_MODE: ${FIPS_MODE}"
echo "   BUILD_ARCH: ${BUILD_ARCH}"
echo "   ENVOY_DOCKER_BUILD_DIR: ${ENVOY_DOCKER_BUILD_DIR}"
echo "   ENVOY_DOCKER_OPTIONS: ${ENVOY_DOCKER_OPTIONS}"
echo "   SSH_AUTH_SOCK: ${SSH_AUTH_SOCK}"
echo "   BAZEL_BUILD_EXTRA_OPTIONS: ${BAZEL_BUILD_EXTRA_OPTIONS[*]}"
echo " "
echo " "

ci_cmd="bazel test --test_output=errors \
 --verbose_failures -c dbg --test_env=ENVOY_IP_TEST_VERSIONS=v4only \
 ${ENVOY_TEST_LABEL}";

if [ ${#BAZEL_BUILD_EXTRA_OPTIONS[@]} -gt 0  ]; then
  ci_cmd="BAZEL_BUILD_EXTRA_OPTIONS='${BAZEL_BUILD_EXTRA_OPTIONS[*]}' $ci_cmd"
fi;

cd "${BASE_ENVOY_DIR}" || exit;
./ci/run_envoy_docker.sh "${ci_cmd}";
cd "${OSS_SOURCE}" || exit;
