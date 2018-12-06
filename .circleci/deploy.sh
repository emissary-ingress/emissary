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
    echo "Skipping deploy for untagged revision."
    exit
fi

VERSION=${TAG}
VERSION_URL=$(python -c "import sys, urllib; print urllib.quote(\"${VERSION}\")")

export AWS_ACCESS_KEY_ID=$DEPLOY_KEY_ID
export AWS_SECRET_ACCESS_KEY=$DEPLOY_KEY

BINS="apictl"

for BIN in ${BINS}; do
    DESTINATION=${BIN}/${VERSION}/$(go env GOOS)/$(go env GOARCH)/${BIN}
    aws s3 cp --acl public-read ${BIN} s3://datawire-static-files/${DESTINATION}
    echo "Uploaded ${BIN} to ${DESTINATION}"
done
