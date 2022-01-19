#!/bin/bash

# some executables
EXE_KUBECTL=${KUBECTL:-$HOME/bin/kubectl}
EXE_HELM2=${HELM2:-$HOME/bin/helm2}
EXE_HELM3=${HELM3:-$HOME/bin/helm}
EXE_KUBERNAUT=${KUBERNAUT:-$HOME/bin/kubernaut}

#######################################################################################################

alias echo_on="{ set -x; }"
alias echo_off="{ set +x; } 2>/dev/null"

RED='\033[1;31m'
GRN='\033[1;32m'
YEL='\033[1;33m'
BLU='\033[1;34m'
WHT='\033[1;37m'
MGT='\033[1;95m'
CYA='\033[1;96m'
END='\033[0m'
BLOCK='\033[1;47m'

log() { >&2 printf "${BLOCK}>>>${END} $1\n"; }

info() { log "${BLU}$1${END}"; }
highlight() { log "${MGT}$1${END}"; }

failed() {
  if [ -z "$1" ] ; then
    log "${RED}failed!!!${END}"
  else
    log "${RED}$1${END}"
  fi
}

passed() {
  if [ -z "$1" ] ; then
    log "${GRN}done!${END}"
  else
    log "${GRN}$1${END}"
  fi
}

bye() {
  log "${BLU}$1... exiting${END}"
  exit 0
}

warn() { log "${RED}!!! WARNING !!! $1 ${END}"; }

abort() {
  log "${RED}FATAL: $1${END}"
  exit 1
}

command_exists() {
    [ -x "$1" ] || command -v $1 >/dev/null 2>/dev/null
}

replace_env_file() {
  info "Replacing env in $1..."
  [ -f "$1" ] || abort "$1 does not exist"
  envsubst < "$1" > "$2"
}

# checks that a URL is available, with an optional error message
check_url() {
  command_exists curl || abort "curl is not installed"
  curl -L --silent -k --output /dev/null --fail "$1"
}

kill_background() {
  info "(Stopping background job)"
  kill $!
  wait $! 2>/dev/null
}

WAIT_TIMEOUT=60

wait_url() {
  local url="$1"
  i=0
  info "Waiting for $url (max $WAIT_TIMEOUT seconds)"
  until [ $i -gt $WAIT_TIMEOUT ] || check_url $url ; do
    info "... still waiting for $url ($i secs passed)"
    i=$((i+1))
    sleep 1
  done
  [ $i -gt $WAIT_TIMEOUT ] && return 1
  return 0
}

wait_pod_running() {
  command_exists "$EXE_KUBECTL" || abort "no kubectl available in $EXE_KUBECTL"
  i=0
  info "Waiting for pod with $@"
  while [ $i -gt $WAIT_TIMEOUT ] || [ "$($EXE_KUBECTL get po $@ -o jsonpath='{.items[0].status.phase}')" != 'Running' ] ; do
    info "... still waiting ($i secs passed)"
    i=$((i+1))
    sleep 1
  done
  [ $i -gt $WAIT_TIMEOUT ] && return 1
  return 0
}

wait_pod_missing() {
  command_exists "$EXE_KUBECTL" || abort "no kubectl available in $EXE_KUBECTL"
  i=0
  info "Waiting for pod with $@ to disappear"
  while [ $i -gt $WAIT_TIMEOUT ] || [ "$($EXE_KUBECTL get po $@ -o name)" != '' ] ; do
    info "... still waiting ($i secs passed)"
    i=$((i+1))
    sleep 1
  done
  [ $i -gt $WAIT_TIMEOUT ] && return 1
  return 0
}

cleanup () {
  info "Cleaning up..."

  $EXE_KUBECTL delete -f $MANIFESTS_DIR/backend.yaml
  kill_background

  $EXE_HELM3 uninstall ambassador > /dev/null
  $EXE_HELM2 del --purge "ambassador-helm2"

  wait_pod_missing "-l app.kubernetes.io/instance=ambassador" || abort "pod still running"
  passed "helm 3 chart uninstalled"

  wait_pod_missing "-l app.kubernetes.io/instance=ambassador-helm2" || abort "pod still running"
  passed "helm 2 chart uninstalled"

  rm -rf "$VALUES_DIR"
}
