#!bash

set -e -o pipefail

HERE=$(cd $(dirname $0); pwd)

cd "$HERE"

ROOT=$(cd ..; pwd)

python ${HERE}/yfix.py ${HERE}/fixes/ambassador.yfix \
    ${ROOT}/docs/yaml/ambassador/ambassador-rbac.yaml \
    ${HERE}/ambassador-deployment.yaml \
    $1 $2
python ${HERE}/yfix.py ${HERE}/fixes/mounts.yfix \
    ${HERE}/ambassador-deployment.yaml \
    ${HERE}/ambassador-deployment-mounts.yaml
