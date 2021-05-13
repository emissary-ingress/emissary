#!/bin/bash

#### NOTE: this chart is not published by any automation or process yet.
# you can run this script manually to publish this chart
# "soon" we'll start publishing this along with the ambassador chart
# and "eventually" this chart will become the only chart we publish

set -e

CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir (maybe running in zsh?)";  exit 1; }
TOP_DIR=$CURR_DIR/..

# shellcheck source=common.sh
source "$CURR_DIR/common.sh"

#########################################################################################
if ! command -v helm 2> /dev/null ; then
    info "Helm doesn't exist, installing helm"
    curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3
    chmod 700 get_helm.sh
    ./get_helm.sh --version v3.4.1
fi
thisversion=$(grep version charts/emissary-ingress/Chart.yaml | awk ' { print $2 }')
repo_key=
if [[ -n "${REPO_KEY}" ]] ; then
    repo_key="${REPO_KEY}"
elif [[ $version =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] ; then
    repo_key=ambassador
else
    repo_key=ambassador-dev
fi
s3url=https://s3.amazonaws.com/datawire-static-files/${repo_key}/

info "Pushing Helm Chart"
helm package $TOP_DIR

# Get name of package
export CHART_PACKAGE=$(ls *.tgz)

curl -o tmp.yaml -k -L ${s3url}index.yaml


if [[ $(grep -c "version: $thisversion" tmp.yaml || true) != 0 ]]; then
	failed "Chart version $thisversion is already in the index"
	exit 1
fi

helm repo index . --url ${s3url} --merge tmp.yaml

if [ -z "$AWS_BUCKET" ] ; then
    AWS_BUCKET=datawire-static-files
fi

[ -n "$AWS_ACCESS_KEY_ID"     ] || abort "AWS_ACCESS_KEY_ID is not set"
[ -n "$AWS_SECRET_ACCESS_KEY" ] || abort "AWS_SECRET_ACCESS_KEY is not set"

info "Pushing chart to S3 bucket $AWS_BUCKET"
for f in "$CHART_PACKAGE" "index.yaml" ; do
  aws s3api put-object \
    --bucket "$AWS_BUCKET" \
    --key "${repo_key}/$f" \
    --body "$f" && passed "... ${repo_key}/$f pushed"
done

info "Cleaning up..."
rm tmp.yaml index.yaml "$CHART_PACKAGE"

exit 0
