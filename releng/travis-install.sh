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

KUBECTL_VERSION=1.10.2

curl -LO https://storage.googleapis.com/kubernetes-release/release/v${KUBECTL_VERSION}/bin/linux/amd64/kubectl
chmod +x kubectl
mv kubectl ~/bin/kubectl

pip install -q -r dev-requirements.txt
pip install -q -r ambassador/requirements.txt
npm install gitbook-cli netlify-cli

if [[ `which helm` == "" ]]; then
  curl https://storage.googleapis.com/kubernetes-helm/helm-v2.9.1-linux-amd64.tar.gz | tar xz
  chmod +x linux-amd64/helm
  sudo mv linux-amd64/helm /usr/local/bin/
  rm -rf linux-amd64
fi

# Initialize helm for indexing use.
helm init --client-only
