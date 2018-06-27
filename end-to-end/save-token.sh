#!/bin/sh

set -e
set -o pipefail

ROOT=$(cd $(dirname $0); pwd)

source ${ROOT}/utils.sh
source ${ROOT}/kubernaut_utils.sh

KUBERNAUT="$ROOT/kubernaut"

get_kubernaut

"$KUBERNAUT" set-token "$1"
