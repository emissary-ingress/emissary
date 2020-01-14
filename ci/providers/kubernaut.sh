#!/bin/bash

CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir (maybe running in zsh?)";  exit 1; }
TOP_DIR="$CURR_DIR/../.."

# shellcheck source=../common.sh
source "$CURR_DIR/../common.sh"

#########################################################################################

CLAIM_NAME="ambassador-chart-${USER}-$(uuidgen)"
CLAIM_FILENAME="$HOME/kubernaut-claim.txt"

DEV_KUBECONFIG="$HOME/.kube/${CLAIM_NAME}.yaml"

KUBERNAUT_CONF="$CURR_DIR/kconf.b64"

#########################################################################################

[ -f "$KUBERNAUT_CONF" ] || abort "no kubernaut conf file found at $KUBERNAUT_CONF"

case $1 in

setup)
  info "Creating kubernaut config..."
  base64 -d < "$KUBERNAUT_CONF" | ( cd ~ ; tar xzf - )
  echo "$CLAIM_NAME" > "$CLAIM_FILENAME"
  ;;

create)
  info "Removing any previous claim for $CLAIM_NAME..."
  kubernaut claims delete "$CLAIM_NAME"

  info "Creating a kubernaut cluster for $CLAIM_NAME..."
  kubernaut claims create --name "$CLAIM_NAME" --cluster-group main || abort "could not claim $CLAIM_NAME"

  info "Doing a quick sanity check on that cluster..."
  kubectl --kubeconfig "$DEV_KUBECONFIG" -n default get service kubernetes || \
    abort "kubernaut was not able to create a valid kubernetes cluster"

  info "kubernaut cluster created"
  ;;

delete)
  info "Releasing kubernaut claim..."
  kubernaut claims delete "$(cat $CLAIM_FILENAME)"
  ;;

get-kubeconfig)
  echo "$HOME/.kube/$(cat $CLAIM_FILENAME).yaml"
  ;;

esac

