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

dsum() {
    local exe=${DIR}/../tools/bin/dsum
    if ! test -f "$exe"; then
        make -C "$DIR/.." tools/bin/dsum
    fi
    "$exe" "$@"
}

msg2() {
    printf "${BLU}  -> ${GRN}%s${END}\n" "$*" >&2
}

panic() {
    printf 'panic: %s\n' "$*" >&2
    exit 1
}

# Usage: build_builder_base [--stage1-only]
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
#    - `Dockerfile.base` changes
#    - `requirements.txt` changes
#    - Enough time has passed (The base only has external/third-party
#      dependencies, and most of those dependencies are not pinned by
#      version, so we rebuild periodically to make sure we don't fall too
#      far behind and then get surprised when a rebuild is required for
#      Dockerfile changes.)  We have defined "enough time" as a few days.
#      See the variable "build_every_n_days" below.
#
#   The base theory of operation is that we generate a Docker tag name that
#   is essentially the tuple
#     (rounded_timestamp, hash(Dockerfile.base), hash(requirements.txt)
#   then check that tag for existence/pullability using `docker run --rm
#   --entrypoint=true`; and build it if it doesn't exist and can't be
#   pulled.
#
#   OK, now for a wee bit of complexity.  We want to use `pip-compile` to
#   update `requirements.txt`.  Because of Python-version-conditioned
#   dependencies, we really want to run it with the image's python3, not
#   with the host's python3.  And since we're updating `requirements.txt`,
#   we don't really want the `pip install` to have already been run.  So,
#   we split the base image in to two stages; stage-1 is everything but
#   `COPY requirements.txt` / `pip install -r requirements.txt`, and then
#   stage-2 copies in `requirements.txt` and runs the `pip install`.  In
#   normal operation we just go ahead and build both stages.  But if the
#   `--stage1-only` flag is given (as it is by the `pip-compile`
#   subcommand), then we only build the stage-1, and set the
#   `builder_base_image` variable to that.
build_builder_base() {
    local builder_base_tag_py='
# Someone please rewrite this in portable Bash. Until then, this code
# works on Python 2.7 and 3.5+.

import datetime, hashlib

# Arrange these 2 variables to reduce the likelihood that build_every_n_days
# passes in the middle of a CI workflow; have it happen weekly during the
# weekend.
build_every_n_days = 7  # Periodic rebuild even if Dockerfile does not change
epoch = datetime.datetime(2020, 11, 8, 5, 0) # 1AM EDT on a Sunday

age = int((datetime.datetime.now() - epoch).days / build_every_n_days)
age_start = epoch + datetime.timedelta(days=age*build_every_n_days)

dockerfilehash = hashlib.sha256(open("Dockerfile.base", "rb").read()).hexdigest()
stage1 = "%sx%s-%s" % (age_start.strftime("%Y%m%d"), build_every_n_days, dockerfilehash[:16])

requirementshash = hashlib.sha256(open("requirements.txt", "rb").read()).hexdigest()
stage2 = "%s-%s" % (stage1, requirementshash[:16])

print("stage1_tag=%s" % stage1)
print("stage2_tag=%s" % stage2)
'

    local stage1_tag stage2_tag
    eval "$(cd "$DIR" && python -c "$builder_base_tag_py")" # sets 'stage1_tag' and 'stage2_tag'

    local BASE_REGISTRY="${BASE_REGISTRY:-${DEV_REGISTRY:-${BUILDER_NAME}.local}}"

    local name1="${BASE_REGISTRY}/builder-base:stage1-${stage1_tag}"
    local name2="${BASE_REGISTRY}/builder-base:stage2-${stage2_tag}"

    msg2 "Using stage-1 base ${BLU}${name1}${GRN}"
    if ! (docker image inspect "$name1" || docker pull "$name2") &>/dev/null; then # skip building if the "$name1" already exists
        dsum 'stage-1 build' 3s \
             docker build -f "${DIR}/Dockerfile.base" -t "${name1}" --target builderbase-stage1 "${DIR}"
        if [[ "$BASE_REGISTRY" == "$DEV_REGISTRY" ]]; then
            TIMEFORMAT="     (stage-1 push took %1R seconds)"
            time docker push "$name1"
            unset TIMEFORMAT
        fi
    fi
    if [[ $1 = '--stage1-only' ]]; then
        builder_base_image="$name1" # not local
        return
    fi

    msg2 "Using stage-2 base ${BLU}${name2}${GRN}"
    if ! (docker image inspect "$name2" || docker pull "$name2") &>/dev/null; then # skip building if the "$name2" already exists
        dsum 'stage-2 build' 3s \
             docker build --build-arg=builderbase_stage1="$name1" -f "${DIR}/Dockerfile.base" -t "${name2}" --target builderbase-stage2 "${DIR}"
        if [[ "$BASE_REGISTRY" == "$DEV_REGISTRY" ]]; then
            TIMEFORMAT="     (stage-2 push took %1R seconds)"
            time docker push "$name2"
            unset TIMEFORMAT
        fi
    fi

    builder_base_image="$name2" # not local
}

cmd="${1:-help}"

case "${cmd}" in
    pip-compile)
        build_builder_base --stage1-only
        printf "${GRN}Running pip-compile to update ${BLU}requirements.txt${END}\n"
        docker run --rm -i "$builder_base_image" sh -c 'tar xf - && pip-compile --allow-unsafe -q >&2 && cat requirements.txt' \
               < <(cd "$DIR" && tar cf - requirements.in requirements.txt) \
               > "$DIR/requirements.txt.tmp"
        mv -f "$DIR/requirements.txt.tmp" "$DIR/requirements.txt"
        ;;

    build-builder-base)
        build_builder_base >&2
        echo "${builder_base_image}"
        ;;
    *)
        echo "usage: builder.sh [pip-compile|build-builder-base]"
        exit 1
        ;;
esac
