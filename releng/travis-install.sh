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

GO_VERSION=1.13
HELM_VERSION=2.9.1
KUBECTL_VERSION=1.10.2
KUBERNAUT_VERSION=2018.10.24-d46c1f1
# https://github.com/datawire/kubernaut/tree/d46c1f13bad67dcb247fec869c882b0a98b71560
# 使用个人编译好的 ARM64 版本：https://github.com/z-jingjie/kubernaut/releases/tag/2018.10.24-d46c1f1-arm64

# http://www.ruanyifeng.com/blog/2017/11/bash-set.html
# 根据返回值来判断，一个命令是否运行失败，若脚本发生错误，就终止其执行，等价于 set -e，但不适用于管道命令
set -o errexit
# 遇到不存在的变量则报错，并停止执行，等价于 set -u
set -o nounset
# 在运行结果之前，先输出执行的那一行命令，等价于 set -x
set -o xtrace

printf "== Begin: travis-install.sh ==\n"

# 创建目录，-p 参数表示如果父级目录尚不存在，也进行目录创建
mkdir -p ~/bin
PATH=~/bin:$PATH

# Install kubectl
# 使用 cURL 下载 kubectl 至 ~/bin/kubectl，且跟随链接的重定向
curl -L -o ~/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/v${KUBECTL_VERSION}/bin/linux/amd64/kubectl
chmod +x ~/bin/kubectl

# Install helm
# 下载并解压 gzip 压缩包至 ...
curl -L https://storage.googleapis.com/kubernetes-helm/helm-v${HELM_VERSION}-linux-amd64.tar.gz | tar -x -z -O linux-amd64/helm > ~/bin/helm
chmod +x ~/bin/helm
# Helm 是 k8s 的包管理器，由客户端命令行工具 helm 和服务端 Tiller 组成，这里不启动 Tiller
helm init --client-only # Initialize helm for indexing use

# Install kubernaut
# 一个临时的 k8s 集群工具，面向开发用途
curl -L -o ~/bin/kubernaut http://releases.datawire.io/kubernaut/${KUBERNAUT_VERSION}/linux/amd64/kubernaut
chmod +x ~/bin/kubernaut

# Install Go
# gimme 是一个安装 Go 的 Shell 脚本
gimme ${GO_VERSION}
# 使用最新版本的 Go
source ~/.gimme/envs/latest.env

# Install awscli
sudo pip install awscli

# Configure kubernaut
# 解码 kconf.b64 文件中的 base64 编码内容，通过管道输出到 list shell 环境中，进入用户根目录，把解码后的数据解压出来
# https://askubuntu.com/questions/1151909/what-does-the-hyphen-mean-in-tar-xzf
base64 -d < kconf.b64 | ( cd ~ ; tar xzf - )
# Grab a kubernaut cluster
CLAIM_NAME=kat-${USER}-$(uuidgen)
DEV_KUBECONFIG=~/.kube/${CLAIM_NAME}.yaml
echo $CLAIM_NAME > ~/kubernaut-claim.txt
kubernaut claims delete ${CLAIM_NAME}
kubernaut claims create --name ${CLAIM_NAME} --cluster-group main
# Do a quick sanity check on that cluster
kubectl --kubeconfig ${DEV_KUBECONFIG} -n default get service kubernetes
# Tell test-warn.sh that, yes, it's OK if it has its way with the cluster
touch .skip_test_warning
# Set up a registry in that cluster
KUBECONFIG=${DEV_KUBECONFIG} go run ./cmd/k8sregistryctl up --storage=hostPath

printf "== End:   travis-install.sh ==\n"
