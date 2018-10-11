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
set -o verbose

GH_TOKEN="${GH_TOKEN:?not set}"
DOC_ROOT="docs"
TARGET_BRANCH="master"
CONTENT_DIR="/tmp/getambassador.io/content"
STATIC_DIR="/tmp/getambassador.io/static"

rm -rf /tmp/getambassador.io
git clone --single-branch -b ${TARGET_BRANCH} https://d6e-automaton:${GH_TOKEN}@github.com/datawire/getambassador.io.git /tmp/getambassador.io

cd docs
cp -R yaml ${STATIC_DIR}
cp doc-links.yml ${STATIC_DIR}
find . \
    -not \( -path ./node_modules -prune \) \
    -not \( -path ./_book -prune \) \
    -name \*.md \
    -exec cp --parent '{}' ${CONTENT_DIR} \;
cd -

cd ${CONTENT_DIR}/..
git add -A
git commit -m "docs updated from datawire/ambassador"
git push origin ${TARGET_BRANCH}
cd -