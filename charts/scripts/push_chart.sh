#!/bin/bash

set -e

CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir (maybe running in zsh?)";  exit 1; }
TOP_DIR=$CURR_DIR/..

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

if [[ -z "${CHART_NAME}" ]] ; then
    abort "Need to specify the chart you wish to publish"
fi
chart_dir="${TOP_DIR}/${CHART_NAME}"

if [[ ! -d "${chart_dir}" ]] ; then
    abort "${chart_dir} is not a directory"
fi

#########################################################################################
if ! command -v helm 2> /dev/null ; then
    info "Helm doesn't exist, installing helm"
    curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3
    chmod 700 get_helm.sh
    ./get_helm.sh --version v3.4.1
    rm -f get_helm.sh
fi
if [[ ${CHART_VERSION:-} != v7.* ]]; then
    abort "CHART_VERSION must be set to a 'v7.*' string"
fi
thisversion=${CHART_VERSION#v}

repo_key=
if [[ -n "${REPO_KEY}" ]] ; then
    repo_key="${REPO_KEY}"
elif [[ $thisversion =~ ^[0-9]+\.[0-9]+\.[0-9]+(-ea)?$ ]] ; then
    repo_key=charts
else
    repo_key=charts-dev
fi
if [ -z "$AWS_S3_BUCKET" ] ; then
    AWS_S3_BUCKET=datawire-static-files
fi

repo_url=https://s3.amazonaws.com/${AWS_S3_BUCKET}/${repo_key}/

rm -f ${chart_dir}/*.tgz
info "Pushing Helm Chart"
helm package --destination $chart_dir $chart_dir

# Get name of package
export CHART_PACKAGE=$(ls ${chart_dir}/*.tgz)

curl -o ${chart_dir}/tmp.yaml -k -L ${repo_url}index.yaml
chart_name=`basename ${chart_dir}`
if [[ $thisversion =~ ^[0-9]+\.[0-9]+\.[0-9]+$  ]] && [[ $(grep -c "${chart_name}-$thisversion\.tgz$" ${chart_dir}/tmp.yaml || true) != 0 ]]; then
	failed "Chart version $thisversion is already in the index"
	exit 1
fi


[ -n "$AWS_ACCESS_KEY_ID"     ] || abort "AWS_ACCESS_KEY_ID is not set"
[ -n "$AWS_SECRET_ACCESS_KEY" ] || abort "AWS_SECRET_ACCESS_KEY is not set"

info "Pushing chart to S3 bucket $AWS_S3_BUCKET"
for f in "$CHART_PACKAGE" ; do
    fname=`basename $f`
    echo "pushing ${repo_key}/$fname"
    aws s3api put-object \
        --bucket "$AWS_S3_BUCKET" \
        --key "${repo_key}/$fname" \
        --body "$f" && passed "... ${repo_key}/$fname pushed"
done

info "Cleaning up..."
echo
rm ${chart_dir}/tmp.yaml "$CHART_PACKAGE"

if [[ `basename ${chart_dir}` != emissary-ingress ]] ; then
    info "This script only publishes release for the emissary-ingress chart, skipping publishing git release for ${chart_dir}"
    exit 0
fi

exit 0
