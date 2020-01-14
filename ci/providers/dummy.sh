#!/bin/bash

CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir (maybe running in zsh?)";  exit 1; }
TOP_DIR="$CURR_DIR/../.."

# shellcheck source=../common.sh
source "$CURR_DIR/../common.sh"

#########################################################################################

case $1 in
create)
  info "Skipping cluster creating..."
  ;;

delete)
  info "Skipping cluster destruction..."
  ;;

get-kubeconfig)
  if [ -n "$KUBECONFIG" ] ; then
    echo "$KUBECONFIG"
  elif [ -n "$DEV_KUBECONFIG" ] ; then
    echo "$DEV_KUBECONFIG"
  fi
  ;;

esac

