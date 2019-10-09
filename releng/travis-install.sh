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

KUBECTL_VERSION=1.10.2
HELM_VERSION=2.9.1
GO_VERSION=1.13

set -o errexit
set -o nounset
set -o xtrace

printf "== Begin: travis-install.sh ==\n"

mkdir -p ~/bin

# Set up for Kubernaut.
base64 -d < kconf.b64 | ( cd ~ ; tar xzf - )

curl -L -o ~/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/v${KUBECTL_VERSION}/bin/linux/amd64/kubectl
chmod +x ~/bin/kubectl

curl -L https://storage.googleapis.com/kubernetes-helm/helm-v${HELM_VERSION}-linux-amd64.tar.gz | tar -x -z -O linux-amd64/helm > ~/bin/helm
chmod +x ~/bin/helm
helm init --client-only # Initialize helm for indexing use

gimme ${GO_VERSION}

pip install -q -r dev-requirements.txt
pip install -q -r python/requirements.txt

printf "== End:   travis-install.sh ==\n"
