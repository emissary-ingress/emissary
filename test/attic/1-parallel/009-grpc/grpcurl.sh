#!/usr/bin/env bash

# set -x

PROGRAM="$(basename $0)"
IMAGE="docker.io/datawire/grpcurl"
VERSION="latest"

# -it
docker run \
  --rm \
  --network host \
  --volume $(pwd):/home/user/work:ro \
  --workdir /home/user/work \
  -e "COMMAND=${COMMAND}" \
  -e HOST_USER_ID=$(id -u) \
  "$IMAGE:$VERSION" "$@" < /dev/null
