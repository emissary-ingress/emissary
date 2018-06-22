#!bash

set -e -o pipefail

HERE=$(cd $(dirname $0); pwd)

cd "$HERE"

ROOT=$(cd ..; pwd)

AMBASSADOR_IMAGE="$1"
STATSD_IMAGE="$2"

# First start with ambassador-rbac.yaml and edit it to be the 
# base ambassador-deployments.yaml that e2e needs.
python ${HERE}/yfix.py ${HERE}/fixes/ambassador.yfix \
    ${ROOT}/docs/yaml/ambassador/ambassador-rbac.yaml \
    ${HERE}/ambassador-deployment.yaml \
    "$AMBASSADOR_IMAGE" "$STATSD_IMAGE"

# Next take that ambassador-deployment.yaml and add some 
# certificate mountpoints and such.
python ${HERE}/yfix.py ${HERE}/fixes/mounts.yfix \
    ${HERE}/ambassador-deployment.yaml \
    ${HERE}/ambassador-deployment-mounts.yaml

# Finally, fix up service.yaml for the websocket test to
# have the right Ambassador image. We'll need to extend 
# this as more tests rely on Forge.
python ${HERE}/yfix.py ${HERE}/fixes/service-yaml.yfix \
    ${HERE}/011-websocket/ambassador/service-src.yaml \
    ${HERE}/011-websocket/ambassador/service.yaml \
    "$AMBASSADOR_IMAGE"

