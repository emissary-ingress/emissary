#!/bin/bash

set -e

CURR_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
[ -d "$CURR_DIR" ] || { echo "FATAL: no current dir (maybe running in zsh?)";  exit 1; }
TOP_DIR=$CURR_DIR/..

# shellcheck source=common.sh
source "$CURR_DIR/common.sh"

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
thisversion=$(grep version ${chart_dir}/Chart.yaml | awk '{ print $2 }')

repo_key=
if [[ -n "${REPO_KEY}" ]] ; then
    repo_key="${REPO_KEY}"
elif [[ $thisversion =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] ; then
    repo_key=ambassador
else
    repo_key=ambassador-dev
fi
repo_url=https://s3.amazonaws.com/datawire-static-files/${repo_key}/

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

helm repo index ${chart_dir} --url ${repo_url} --merge ${chart_dir}/tmp.yaml

if [ -z "$AWS_BUCKET" ] ; then
    AWS_BUCKET=datawire-static-files
fi

[ -n "$AWS_ACCESS_KEY_ID"     ] || abort "AWS_ACCESS_KEY_ID is not set"
[ -n "$AWS_SECRET_ACCESS_KEY" ] || abort "AWS_SECRET_ACCESS_KEY is not set"

info "Pushing chart to S3 bucket $AWS_BUCKET"
for f in "$CHART_PACKAGE" "${chart_dir}/index.yaml" ; do
    fname=`basename $f`
    echo "pushing ${repo_key}/$fname"
    aws s3api put-object \
        --bucket "$AWS_BUCKET" \
        --key "${repo_key}/$fname" \
        --body "$f" && passed "... ${repo_key}/$fname pushed"
done

info "Cleaning up..."
echo
rm ${chart_dir}/tmp.yaml ${chart_dir}/index.yaml "$CHART_PACKAGE"

if [[ `basename ${chart_dir}` != ambassador ]] ; then
    info "This script only publishes release for the ambassador chart, skipping publishing git release for ${chart_dir}"
    exit 0
fi

if [[ $thisversion =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] && [[ -n "${PUBLISH_GIT_RELEASE}" ]]; then
    if [[ -z "${CIRCLE_SHA1}" ]] ; then
        echo "CIRCLE_SHA1 not set"
        exit 1
    fi
    if [[ -z "${GH_RELEASE_TOKEN}" ]] ; then
        echo "GH_RELEASE_TOKEN not set"
        exit 1
    fi
    tag="chart-v${thisversion}"
    export CHART_VERSION=${thisversion}
    title=`envsubst < ${chart_dir}/RELEASE_TITLE.tpl`
    repo_full_name="emissary-ingress/emissary"
    token="${GH_RELEASE_TOKEN}"
    description=`envsubst < ${chart_dir}/RELEASE.tpl | awk '{printf "%s\\\n", $0}'`
    in_changelog=false
    while IFS= read -r line ; do
        if ${in_changelog} ; then
            if [[ "${line}" =~ "## v" ]] ; then
                break
            fi
            if [[ -n "${line}" ]] ; then
                description="${description}\\n${line}"
            fi
        fi
        if [[ "${line}" =~ "## v${chart_version}" ]] ; then
            in_changelog=true
        fi

    done < ${chart_dir}/CHANGELOG.md

    generate_post_data()
    {
        cat <<EOF
{
  "tag_name": "$tag",
  "name": "$title",
  "body": "${description}",
  "draft": false,
  "prerelease": false,
  "target_commitish": "${CIRCLE_SHA1}"
}
EOF
    }
    curl -H "Authorization: token ${token}" --data "$(generate_post_data)" "https://api.github.com/repos/$repo_full_name/releases"
fi

exit 0
