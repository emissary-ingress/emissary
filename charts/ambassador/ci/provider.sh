#!/bin/bash

CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir (maybe running in zsh?)";  exit 1; }
TOP_DIR=$CURR_DIR/..

# shellcheck source=common.sh
source "$CURR_DIR/common.sh"

#########################################################################################

EXE_PROVIDER="$CURR_DIR/providers/$PROVIDER.sh"

#########################################################################################

[ -x "$EXE_PROVIDER" ] || abort "no kubernetes provider found in $EXE_PROVIDER"

case $1 in
setup)
  info "Setting up $PROVIDER"
  exec $EXE_PROVIDER setup
  ;;

create)
  info "Creating $PROVIDER cluster"
  exec $EXE_PROVIDER create
  ;;

delete)
  info "Destroying $PROVIDER cluster"
  exec $EXE_PROVIDER delete
  ;;

get-kubeconfig)
  exec $EXE_PROVIDER get-kubeconfig
  ;;

esac
