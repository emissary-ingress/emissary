#!/bin/bash

CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir (maybe running in zsh?)";  exit 1; }
TOP_DIR="$CURR_DIR/.."

# shellcheck source=common.sh
source "$CURR_DIR/common.sh"

#########################################################################################################

HELM_VERSION=3.0.2
HELM2_VERSION=2.16.1
KUBECTL_VERSION=1.15.3
KUBERNAUT_VERSION=2018.10.24-d46c1f1

#########################################################################################################

mkdir -p "$HOME/bin"
export PATH=$HOME/bin:$PATH

if ! command_exists "$EXE_KUBECTL" ; then
  info "Installing kubectl..."
  curl -L -o "$EXE_KUBECTL" https://storage.googleapis.com/kubernetes-release/release/v${KUBECTL_VERSION}/bin/linux/amd64/kubectl
  chmod +x "$EXE_KUBECTL"
fi

if ! command_exists "$EXE_HELM3" ; then
  info "Installing Helm 3..."
  curl -L https://get.helm.sh/helm-v${HELM_VERSION}-linux-amd64.tar.gz | tar -x -z -O linux-amd64/helm > $EXE_HELM3
  chmod +x "$EXE_HELM3"
fi


if ! command_exists "$EXE_HELM2" ; then
  info "Installing Helm 2..."
  curl -L https://storage.googleapis.com/kubernetes-helm/helm-v${HELM2_VERSION}-linux-amd64.tar.gz | tar -x -z -O linux-amd64/helm > $EXE_HELM2
  chmod +x "$EXE_HELM2"
fi

if ! command_exists "$EXE_KUBERNAUT" ; then
  info "Installing Kubernaut..."
  curl -L -o "$EXE_KUBERNAUT" http://releases.datawire.io/kubernaut/${KUBERNAUT_VERSION}/linux/amd64/kubernaut
  chmod +x "$EXE_KUBERNAUT"
fi

if ! command_exists awscli ; then
  info "Installing awscli..."
  sudo pip install awscli
fi

