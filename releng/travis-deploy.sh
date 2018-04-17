#!/usr/bin/env bash
set -o errexit
set -o nounset

branch="${1:?branch name is not set}"
commit="${2:?commit hash is not set}"
version="${3:-version_undefied}"

# There is only one real rule in this branch: If 'version' is not set then we are performing an unstable release
# otherwise consider this a 'stable' release.
#
# - An unstable release pushes a docker image into a docker repository with a tag that corresponds to the commit.
# - A stable release pushes a new docker tag into a docker repository for an extant docker image.
#
# After a 'stable release' is performed the following additional operations are performed:
#
# - push app.json metadata into Scout S3 bucket.

if [[ "${version}" != "version_undefined" ]]; then

    # Perform a release by pulling the Docker image associated with the tag which should already be committed
    DOCKER_REGISTRY="quay.io/datawire"

    echo ${DOCKER_PASSWORD} | docker login -u="${DOCKER_USERNAME}" --password-stdin quay.io

    printf "Start full stable release operations"
fi