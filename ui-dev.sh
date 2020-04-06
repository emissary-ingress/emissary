#!/bin/bash

SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ]; do # resolve $SOURCE until the file is no longer a symlink
  DIR="$( cd -P "$( dirname "$SOURCE" )" >/dev/null 2>&1 && pwd )"
  SOURCE="$(readlink "$SOURCE")"
  [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE" # if $SOURCE was a relative symlink, we need to resolve it relative to the path where the symlink file was located
done
DIR="$( cd -P "$( dirname "$SOURCE" )" >/dev/null 2>&1 && pwd )"

IP=$(kubectl get -n ambassador service ambassador -o 'go-template={{range .status.loadBalancer.ingress}}{{print .ip "\n"}}{{end}}')


ACTUAL_COMMIT=$(git -C ${DIR}/../ambassador rev-parse HEAD)
PINNED_COMMIT=$(cat ${DIR}/ambassador.commit)

if [ "$1" != "force" ] && [ "${ACTUAL_COMMIT}" != "${PINNED_COMMIT}" ]; then
    printf "Warning, your ambassador checkout is not at the pinned commit!\n"
    printf "Run the following command to sync:\n\n"
    printf "  (cd ../ambassador/ && git fetch && git checkout $(cat ../apro/ambassador.commit))\n\n"
    printf "Alternatively, you can use the force argument to override this check:\n\n"
    printf "  ./ui-dev.sh force\n\n"
    exit 1
fi

DEV_WEBUI_SNAPSHOT_HOST=${IP} \
DEV_WEBUI_DIR=${DIR}/cmd/amb-sidecar/webui/bindata \
DEV_AES_HTTP_PORT=8501 \
DEV_WEBUI_PORT=9000 \
POD_NAMESPACE=ambassador \
AMBASSADOR_NAMESPACE=ambassador \
  go run ./cmd/ambassador amb-sidecar
