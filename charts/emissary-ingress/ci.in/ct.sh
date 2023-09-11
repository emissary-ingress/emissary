#!/bin/sh

set -ex

helm repo add ambassador-agent https://s3.amazonaws.com/datawire-static-files/charts || helm repo update

ct "$@"
