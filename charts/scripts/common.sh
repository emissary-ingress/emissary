#!/bin/bash

RED='\033[1;31m'
GRN='\033[1;32m'
BLU='\033[1;34m'
END='\033[0m'
BLOCK='\033[1;47m'

log() { >&2 printf "${BLOCK}>>>${END} $1\n"; }

info() { log "${BLU}$1${END}"; }

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

abort() {
  log "${RED}FATAL: $1${END}"
  exit 1
}
