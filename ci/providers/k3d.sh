#!/bin/bash

CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir (maybe running in zsh?)";  exit 1; }
TOP_DIR="$CURR_DIR/../.."

# shellcheck source=../common.sh
source "$CURR_DIR/../common.sh"

#########################################################################################

K3D_CLUSTER_NAME="k3s-default"

#########################################################################################

case $1 in
setup)
  #	info "Making sure Docker is not running..."
  #	sudo systemctl stop docker || /bin/true
  #
  #	info "Installing a more modern Docker version..."
  #	curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add -
  #	sudo add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu `lsb_release -cs` stable"
  #	sudo apt-get update
  #	sudo apt-get -y -o Dpkg::Options::="--force-confnew" install docker-ce
  #
  #	info "Re-enabling the Docker service..."
  #	sudo systemctl enable --now docker

  if ! command_exists k3d ; then
    info "Installing k3d"
    curl -s https://raw.githubusercontent.com/rancher/k3d/master/install.sh | bash
  fi
  ;;

create)
  info "Creating k3d cluster..."
  k3d create --wait 60 --name="$K3D_CLUSTER_NAME"
  KUBECONFIG=$($0 get-kubeconfig)
  [ -n "$KUBECONFIG" ] || abort "could not obtain a valid KUBECONFIG from k3d"

  info "Showing some k3d cluster info:"
  kubectl --kubeconfig="$KUBECONFIG" cluster-info
  ;;

delete)
  info "Destroying k3d cluster..."
  k3d delete --name="$K3D_CLUSTER_NAME"
  ;;

get-kubeconfig)
  k3d get-kubeconfig --name="$K3D_CLUSTER_NAME"
  ;;

esac

