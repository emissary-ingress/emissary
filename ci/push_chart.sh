#!/bin/bash


CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir (maybe running in zsh?)";  exit 1; }
TOP_DIR=$CURR_DIR/..

# shellcheck source=common.sh
source "$CURR_DIR/common.sh"

#########################################################################################

if [ -z "$TRAVIS_TAG" ]  ; then
  info "No TRAVIS_TAG in environment: the chart will not be pushed..."
  exit 0
fi

if [ -z "$PUSH_CHART" ] || [ "$PUSH_CHART" = "false" ] ; then
  info "PUSH_CHART is undefined (or defined as false) in environment: the chart will not be pushed..."
  exit 0
fi

info "Pushing Helm Chart"
helm package $TOP_DIR

# Get name of package
export CHART_PACKAGE=$(ls *.tgz)

curl -o tmp.yaml -k -L https://getambassador.io/helm/index.yaml

helm repo index . --url https://getambassador.io/helm --merge tmp.yaml

info "Pushing chart to S3 bucket"
aws s3api put-object \
  --bucket datawire-static-files \
  --key ambassador/$CHART_PACKAGE \
  --body $CHART_PACKAGE

aws s3api put-object \
  --bucket datawire-static-files \
  --key ambassador/index.yaml \
  --body index.yaml

info "Cleanup"
rm tmp.yaml index.yaml $CHART_PACKAGE

info "Pushing Helm Chart"

