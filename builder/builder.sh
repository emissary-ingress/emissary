#!/usr/bin/env bash

# Choose colors carefully. If they don't work on both a black
# background and a white background, pick other colors (so white,
# yellow, and black are poor choices).
RED=$'\033[1;31m'
GRN=$'\033[1;32m'
BLU=$'\033[1;34m'
CYN=$'\033[1;36m'
END=$'\033[0m'

set -e

SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ]; do # resolve $SOURCE until the file is no longer a symlink
    DIR="$( cd -P "$( dirname "$SOURCE" )" >/dev/null 2>&1 && pwd )"
    SOURCE="$(readlink "$SOURCE")"
    [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE" # if $SOURCE was a relative symlink, we need to resolve it relative to the path where the symlink file was located
done
DIR="$( cd -P "$( dirname "$SOURCE" )" >/dev/null 2>&1 && pwd )"

msg2() {
    printf "${BLU}  -> ${GRN}%s${END}\n" "$*" >&2
}

panic() {
    printf 'panic: %s\n' "$*" >&2
    exit 1
}

# Usage: build_builder
# Effects:
#   1. Set the `builder_base_image` variable in the parent scope
#   2. Ensure that the `$builder_base_image` Docker image exists (pulling
#      it or building it if it doesn't).
#   3. (If $DEV_REGISTRY is set AND we built the image) push the
#      `$builder_base_image` Docker image.
#
# Description:
#
#   Rebuild (and push if DEV_REGISTRY is set) the builder's base image if
#    - `docker/base-python/Dockerfile` changes
#    - Enough time has passed (The base only has external/third-party
#      dependencies, and most of those dependencies are not pinned by
#      version, so we rebuild periodically to make sure we don't fall too
#      far behind and then get surprised when a rebuild is required for
#      Dockerfile changes.)  We have defined "enough time" as a few days.
#      See the variable "build_every_n_days" below.
#
#   The base theory of operation is that we generate a Docker tag name that
#   is essentially the tuple
#       (rounded_timestamp, hash("docker/base-python/Dockerfile"))
#   then check that tag for existence/pullability using
#   `docker inspect || docker pull`; and build it if it doesn't exist
#   and can't be pulled.
build_builder_base() {
    local builder_base_tag_py='
import datetime, hashlib

# Arrange these 2 variables to reduce the likelihood that build_every_n_days
# passes in the middle of a CI workflow; have it happen weekly during the
# weekend.
build_every_n_days = 7  # Periodic rebuild even if Dockerfile does not change
epoch = datetime.datetime(2020, 11, 8, 5, 0) # 1AM EDT on a Sunday

age = int((datetime.datetime.now() - epoch).days / build_every_n_days)
age_start = epoch + datetime.timedelta(days=age*build_every_n_days)

dockerfilehash = hashlib.sha256(open("docker/base-python/Dockerfile", "rb").read()).hexdigest()
stage1 = "%sx%s-%s" % (age_start.strftime("%Y%m%d"), build_every_n_days, dockerfilehash[:16])

print("stage1_tag=%s" % stage1)
'

    local stage1_tag stage2_tag
    eval "$(python3 -c "$builder_base_tag_py")" # sets 'stage1_tag' and 'stage2_tag'

    local name1="${BASE_REGISTRY}/builder-base:stage1-${stage1_tag}"

    msg2 "Using stage-1 base ${BLU}${name1}${GRN}"
    if ! (docker image inspect "$name1" || docker pull "$name1") &>/dev/null; then # skip building if the "$name1" already exists
        tools/bin/dsum 'stage-1 build' 3s \
             docker build -t "${name1}" docker/base-python
        if [[ "$BASE_REGISTRY" == "$DEV_REGISTRY" ]]; then
            TIMEFORMAT="     (stage-1 push took %1R seconds)"
            time docker push "$name1"
            unset TIMEFORMAT
        fi
    fi
    builder_base_image="$name1" # not local
}

cmd="${1:-help}"

case "${cmd}" in
    build-builder-base)
        build_builder_base >&2
        echo "${builder_base_image}"
        ;;
    *)
        echo "usage: builder.sh [build-builder-base]"
        exit 1
        ;;
esac
