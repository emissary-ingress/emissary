#!bash

# Copyright 2018 Datawire. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License

set -e -o pipefail

HERE=$(cd $(dirname $0); pwd)

cd "$HERE"

ROOT=$(cd ..; pwd)

AMBASSADOR_IMAGE="$1"

# First start with ambassador-rbac.yaml and edit it to be the 
# base ambassador-deployments.yaml that e2e needs.
python ${HERE}/yfix.py ${HERE}/fixes/ambassador.yfix \
    ${ROOT}/docs/yaml/ambassador/ambassador-rbac.yaml \
    ${HERE}/ambassador-deployment.yaml \
    "$AMBASSADOR_IMAGE"

# Next take that ambassador-deployment.yaml and add some 
# certificate mountpoints and such.
python ${HERE}/yfix.py ${HERE}/fixes/mounts.yfix \
    ${HERE}/ambassador-deployment.yaml \
    ${HERE}/ambassador-deployment-mounts.yaml

# Finally, fix up service.yaml for the websocket test to
# have the right Ambassador image. We'll need to extend 
# this as more tests rely on Forge.
python ${HERE}/yfix.py ${HERE}/fixes/service-yaml.yfix \
    ${HERE}/1-parallel/011-websocket/ambassador/service-src.yaml \
    ${HERE}/1-parallel/011-websocket/ambassador/service.yaml \
    "$AMBASSADOR_IMAGE"

