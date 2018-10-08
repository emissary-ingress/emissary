#!/usr/bin/env bash

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

set -o errexit
set -o nounset

GH_TOKEN="${GH_TOKEN:?not set}"
DOC_ROOT="docs"
CONTENT_DIR="/tmp/getambassador.io/content"

git clone --single-branch -b dev/ambassador-853 https://d6e-automaton:${GH_TOKEN}@github.com/datawire/getambassador.io.git /tmp/getambassador.io
mkdir -p ${CONTENT_DIR}

cd docs
cp -R yaml ${CONTENT_DIR}
find . \
    -not \( -path ./node_modules -prune \) \
    -not \( -path ./_book -prune \) \
    -name \*.md \
    -exec cp --parent '{}' ${CONTENT_DIR} \;
cd -

cd ${CONTENT_DIR}/..
git add -A
git commit -m "docs updated from datawire/ambassador"
git push origin dev/ambassador-853
cd -