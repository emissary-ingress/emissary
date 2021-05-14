#!/bin/bash

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

info "Pushing Helm Chart"
helm package $TOP_DIR

# Get name of package
export CHART_PACKAGE=$(ls *.tgz)

curl -o tmp.yaml -k -L https://getambassador.io/helm/index.yaml

thisversion=$(get_chart_version ${TOP_DIR})

if [[ $(grep -c "version: $thisversion" tmp.yaml || true) != 0 ]]; then
    failed "Chart version $thisversion is already in the index"
    exit 1
fi

helm repo index . --url https://getambassador.io/helm --merge tmp.yaml

if [ -z "$AWS_BUCKET" ] ; then
    AWS_BUCKET=datawire-static-files
fi

[ -n "$AWS_ACCESS_KEY_ID"     ] || abort "AWS_ACCESS_KEY_ID is not set"
[ -n "$AWS_SECRET_ACCESS_KEY" ] || abort "AWS_SECRET_ACCESS_KEY is not set"

info "Pushing chart to S3 bucket $AWS_BUCKET"
for f in "$CHART_PACKAGE" "index.yaml" ; do
  aws s3api put-object \
    --bucket "$AWS_BUCKET" \
    --key "ambassador/$f" \
    --body "$f" && passed "... ambassador/$f pushed"
done

info "Cleaning up..."
rm tmp.yaml index.yaml "$CHART_PACKAGE"

if [[ -n "${PUBLISH_GIT_RELEASE}" ]] ; then
    if [[ -z "${CIRCLE_SHA1}" ]] ; then
        echo "CIRCLE_SHA1 not set"
        exit 1
    fi
    if [[ -z "${GH_RELEASE_TOKEN}" ]] ; then
        echo "GH_RELEASE_TOKEN not set"
        exit 1
    fi
    tag="chart-v${thisversion}"
    title="Ambassador Chart ${thisversion}"
    repo_full_name="datawire/ambassador"
    token="${GH_RELEASE_TOKEN}"
    description=$(cat <<-END
## :tada: Ambassador Chart ${thisversion} :tada:

Upgrade Ambassador - https://www.getambassador.io/reference/upgrading#helm.html
View changelog - https://github.com/datawire/ambassador/blob/master/charts/ambassador/CHANGELOG.md

---

END
)
    description=`echo "${description}" | awk '{printf "%s\\\n", $0}'`
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

    done < ${TOP_DIR}/CHANGELOG.md

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
