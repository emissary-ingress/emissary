#!/bin/sh

set -e
set -u

if [ -n $(TRAVIS_TAG) ]; then
  echo "Waiting for authorizarion service deployment..."
  typeset -i cnt=60
  until kubectl rollout status deployment/auth0-service -n datawire | grep "successfully rolled out"; do
    ((cnt=cnt-1)) || exit 1
    sleep 2
  done

  echo "Waiting for ambassador deployment..."
  typeset -i cnt=60
  until kubectl rollout status deployment/ambassador -n datawire | grep "successfully rolled out"; do
    ((cnt=cnt-1)) || exit 1
    sleep 2
  done  
fi
