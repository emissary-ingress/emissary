#!/bin/bash
set -e

SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ]; do # resolve $SOURCE until the file is no longer a symlink
  DIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"
  SOURCE="$(readlink "$SOURCE")"
  [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE" # if $SOURCE was a relative symlink, we need to resolve it relative to the path where the symlink file was located
done
DIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"

SRC_DIR=${DIR}/..

cd ${SRC_DIR}

TAG="$(git describe --exact-match --tags HEAD || true)"

if [ -z "$TAG" ]; then
    echo "Skipping promote for untagged revision."
    exit
fi

VERSION=${TAG}

IMAGE=quay.io/datawire/ambassador-ratelimit:${VERSION}

docker tag $(cat pushed.txt) ${IMAGE}
docker push ${IMAGE}
