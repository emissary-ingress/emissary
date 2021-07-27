#!/usr/bin/env bash

shopt -s expand_aliases

alias echo_on="{ set -x; }"
alias echo_off="{ set +x; } 2>/dev/null"

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
TEST_DATA_DIR=/tmp/test-data/
if [[ -n "${TEST_XML_DIR}" ]] ; then
    TEST_DATA_DIR=${TEST_XML_DIR}
fi

DBUILD=${DIR}/dbuild.sh

now=$(date +"%H%M%S")

# container name of the builder
BUILDER_CONT_NAME=${BUILDER_CONT_NAME:-"bld-${BUILDER_NAME}-${now}"}

# command for running a container (ie, "docker run")
BUILDER_DOCKER_RUN=${BUILDER_DOCKER_RUN:-docker run}

# the name of the Docker network
# note: this is necessary for connecting the builder to a local k3d/microk8s/kind network (ie, for running tests)
BUILDER_DOCKER_NETWORK=${BUILDER_DOCKER_NETWORK:-${BUILDER_NAME}}

# Do this with `eval` so that we properly interpret quotes.
eval "pytest_args=(${PYTEST_ARGS:-})"

msg() {
    printf "${CYN}==> ${GRN}%s${END}\n" "$*" >&2
}

msg2() {
    printf "${BLU}  -> ${GRN}%s${END}\n" "$*" >&2
}

panic() {
    printf 'panic: %s\n' "$*" >&2
    exit 1
}

builder() {
    if ! [ -e docker/builder-base.docker ]; then
        panic "This should not happen: 'docker/builder-base.docker' does not exist"
    fi
    if ! [ -e docker/base-envoy.docker ]; then
        panic "This should not happen: 'docker/base-envoy.docker' does not exist"
    fi
    local builder_base_image envoy_base_image
    builder_base_image=$(cat docker/builder-base.docker)
    envoy_base_image=$(cat docker/base-envoy.docker)
    docker ps --quiet \
           --filter=label=builder \
           --filter=label="$BUILDER_NAME" \
           --filter=label=builderbase="$builder_base_image" \
           --filter=label=envoybase="$envoy_base_image"
}
builder_network() { docker network ls -q -f name="${BUILDER_DOCKER_NETWORK}"; }

builder_volume() { docker volume ls -q -f label=builder; }

declare -a dsynced

dsync() {
    msg2 "Synchronizing... $*"
    TIMEFORMAT="     (sync took %1R seconds)"
    time IFS='|' read -ra dsynced <<<"$(rsync --info=name -aO --blocking-io -e 'docker exec -i' $@ 2> >(fgrep -v 'rsync: failed to set permissions on' >&2) | tr '\n' '|')"
}

dcopy() {
    msg2 "Copying... $*"
    local TIMEFORMAT="     (copy took %1R seconds)"
    time docker cp $@
}

dexec() {
    if [[ -t 0 ]]; then
        flags=-it
    else
        flags=-i
    fi
    docker exec ${flags} $(builder) "$@"
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
        TIMEFORMAT="     (stage-1 build took %1R seconds)"
        time ${DBUILD} -f "${DIR}/Dockerfile.base" -t "${name1}" --target builderbase-stage1 "${DIR}"
        unset TIMEFORMAT
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
        TIMEFORMAT="     (stage-2 build took %1R seconds)"
        time ${DBUILD} --build-arg=builderbase_stage1="$name1" -f "${DIR}/Dockerfile.base" -t "${name2}" --target builderbase-stage2 "${DIR}"
        unset TIMEFORMAT
        if [[ "$BASE_REGISTRY" == "$DEV_REGISTRY" ]]; then
            TIMEFORMAT="     (stage-2 push took %1R seconds)"
            time docker push "$name2"
            unset TIMEFORMAT
        fi
    fi

    builder_base_image="$name2" # not local
}

bootstrap() {
    if [ -z "$(builder_volume)" ] ; then
        docker volume create --label builder
        msg2 "Created docker volume ${BLU}$(builder_volume)${GRN} for caching"
    fi

    if [ -z "$(builder_network)" ]; then
        msg2 "Creating docker network ${BLU}${BUILDER_DOCKER_NETWORK}${GRN}"
        docker network create "${BUILDER_DOCKER_NETWORK}" > /dev/null
    else
        msg2 "Connecting to existing network ${BLU}${BUILDER_DOCKER_NETWORK}${GRN}"
    fi

    if [ -z "$(builder)" ] ; then
        if ! [ -e docker/builder-base.docker ]; then
            panic "This should not happen: 'docker/builder-base.docker' does not exist"
        fi
        if ! [ -e docker/base-envoy.docker ]; then
            panic "This should not happen: 'docker/base-envoy.docker' does not exist"
        fi
        local builder_base_image envoy_base_image
        builder_base_image=$(cat docker/builder-base.docker)
        envoy_base_image=$(cat docker/base-envoy.docker)
        msg2 'Bootstrapping build image'
        TIMEFORMAT="     (builder bootstrap took %1R seconds)"
        time ${DBUILD} \
            --build-arg=envoy="${envoy_base_image}" \
            --build-arg=builderbase="${builder_base_image}" \
            --target=builder \
            ${DIR} -t ${BUILDER_NAME}.local/builder
        unset TIMEFORMAT
        if stat --version | grep -q GNU ; then
            DOCKER_GID=$(stat -c "%g" /var/run/docker.sock)
        else
            DOCKER_GID=$(stat -f "%g" /var/run/docker.sock)
        fi
        if [ -z "${DOCKER_GID}" ]; then
            panic "Unable to determine docker group-id"
        fi

        msg2 'Starting build container...'

        echo_on
        $BUILDER_DOCKER_RUN \
            --name="$BUILDER_CONT_NAME" \
            --network="${BUILDER_DOCKER_NETWORK}" \
            --network-alias="builder" \
            --group-add="${DOCKER_GID}" \
            --detach \
            --rm \
            --volume=/var/run/docker.sock:/var/run/docker.sock \
            --volume="$(builder_volume):/home/dw" \
            ${BUILDER_MOUNTS} \
            --cap-add=NET_ADMIN \
            --label=builder \
            --label="${BUILDER_NAME}" \
            --label=builderbase="$builder_base_image" \
            --label=envoybase="$envoy_base_image" \
            ${BUILDER_PORTMAPS} \
            ${BUILDER_DOCKER_EXTRA} \
            --env=BUILDER_NAME="${BUILDER_NAME}" \
            --env=GOPRIVATE="${GOPRIVATE}" \
            --env=AWS_SECRET_ACCESS_KEY \
            --env=AWS_ACCESS_KEY_ID \
            --env=AWS_SESSION_TOKEN \
            --init \
            --entrypoint=tail ${BUILDER_NAME}.local/builder -f /dev/null > /dev/null
        echo_off

        msg2 "Started build container ${BLU}$(builder)${GRN}"
    fi

    dcopy ${DIR}/builder.sh $(builder):/buildroot
    dcopy ${DIR}/builder_bash_rc $(builder):/home/dw/.bashrc

    # If we've been asked to muck with gitconfig, do it.
    if [ -n "$SYNC_GITCONFIG" ]; then
        dsync "$SYNC_GITCONFIG" $(builder):/home/dw/.gitconfig
    fi
}

module_version() {
    echo MODULE="\"$1\""

    # What version is in docs/yaml/version.yaml?

    BASE_VERSION=

    if [ -f docs/yaml/versions.yml ]; then
        BASE_VERSION=$(grep version: docs/yaml/versions.yml | awk ' { print $2 }')
        if [[ "${BASE_VERSION}" =~ -ea$ ]] ; then
            BASE_VERSION=${BASE_VERSION%-ea}
        fi
    else
        # We have... nothing.
        echo "No base version" >&2
        exit 1
    fi

    # EXTRA_VERSION gets added to BASE_VERSION (below). Start it out empty.
    EXTRA_VERSION=

    # Get a bunch of git info, starting with the branch.
    echo GIT_BRANCH="\"$(git rev-parse --abbrev-ref HEAD)\""

    # The short git commit hash
    echo GIT_COMMIT="\"$(git rev-parse --short HEAD)\""
    # Whether `git add . && git commit` would commit anything (empty=false, nonempty=true)
    if [ -n "$(git status --porcelain)" ]; then
        echo GIT_DIRTY="\"dirty\""
        dirty="yes"
    else
        echo GIT_DIRTY="\"\""
        dirty=""
    fi
    # The _previous_ tag, plus a git delta, like 'v1.13.3-117-g2434c437f'... or, if we're _on_
    # a tag, just something like 'v1.13.3'. Don't allow hotfix tags to appear here, though!
    GIT_DESCRIPTION=$(git describe --tags --match 'v*' --exclude '*-hf.*')
    echo GIT_DESCRIPTION="\"$GIT_DESCRIPTION\""

    # Do we have a '-' in our GIT_DESCRIPTION?
    if [[ ${GIT_DESCRIPTION} =~ - ]]; then
        # Pull out fields from that.
        GIT_VERSION=$(echo $GIT_DESCRIPTION | cut -d- -f1)
        GIT_REST=$(echo $GIT_DESCRIPTION | cut -d- -f2-)
        if [[ ${GIT_REST} =~ ^[a-z] ]] && [[ ${GIT_REST} =~ - ]]; then
            # git describe isn't exactly at an rc or ea tag
            # so let's filter those out when getting the git description
            GIT_DESCRIPTION=$(git describe --tags --match 'v*' --exclude '*-*')
            GIT_VERSION=$(echo $GIT_DESCRIPTION | cut -d- -f1)
            GIT_REST=$(echo $GIT_DESCRIPTION | cut -d- -f2-)
        fi

        # If the first character of GIT_REST is alphabetic, we should be looking
        # at an "-rc" or "-ea" tag or the like, and there should not be another -
        # in it.
        if [[ ${GIT_REST} =~ ^[a-z] ]]; then
            if [[ ${GIT_REST} =~ - ]]; then
                echo "GIT_VERSION $GIT_VERSION is not understood" >&2
                exit 1
            fi

            # Good to go. Remember to put the leading "-" back here.
            EXTRA_VERSION="-${GIT_REST}"
        else
            # GIT_REST should be N-gH, so split it into parts...
            GIT_COUNT=$(echo $GIT_REST | cut -d- -f1)
            GIT_HASH=$(echo $GIT_REST | cut -d- -f2)

            # ...and build EXTRA_VERSION from that.
            EXTRA_VERSION="-dev.${GIT_COUNT}+${GIT_HASH}"
        fi
    else
        # We're on a tag. Does it match our build version?
        if [ "$GIT_DESCRIPTION" != "v$BASE_VERSION" ]; then
            echo "Tag $GIT_DESCRIPTION does not match base version $BASE_VERSION" >&2
            exit 1
        fi

        # All good, use no EXTRA_VERSION stuff here -- but set GIT_VERSION just in
        # case someone wants it?
        GIT_VERSION=$GIT_DESCRIPTION
    fi

    # RELEASE_VERSION is a semver string that we use for tagging things.
    # BUILD_VERSION is a semver string that we build into the images.
    #
    # Neither of these should have a leading 'v'.

    BUILD_VERSION="${BASE_VERSION}${EXTRA_VERSION}"

    if [[ ${BUILD_VERSION} =~ ^v[0-9]+.*$ ]]; then
        BUILD_VERSION=${BUILD_VERSION:1}
    fi

    RELEASE_VERSION=$BUILD_VERSION

    if [ -n "${dirty}" ]; then
        RELEASE_VERSION="${RELEASE_VERSION}-dirty"
    fi

    echo GIT_VERSION="\"${GIT_VERSION}\""
    echo GIT_REST="\"${GIT_REST}\""
    echo BASE_VERSION="\"${BASE_VERSION}\""
    echo EXTRA_VERSION="\"${EXTRA_VERSION}\""
    echo RELEASE_VERSION="\"${RELEASE_VERSION}\""
    echo BUILD_VERSION="\"${BUILD_VERSION}\""
}

sync() {
    name=$1
    sourcedir=$2
    container=$3

    real=$(cd ${sourcedir}; pwd)

    dexec mkdir -p /buildroot/${name}
    if [[ $name == apro ]]; then
        # Don't let 'deleting ambassador' cause the sync to be marked dirty
        dexec sh -c 'rm -rf apro/ambassador'
    fi
    dsync $DSYNC_EXTRA --exclude-from=${DIR}/sync-excludes.txt --delete ${real}/ ${container}:/buildroot/${name}
    summarize-sync $name "${dsynced[@]}"
    if [[ $name == apro ]]; then
        # BusyBox `ln` 1.30.1's `-T` flag is broken, and doesn't have a `-t` flag.
        dexec sh -c 'if ! test -L apro/ambassador; then rm -rf apro/ambassador && ln -s ../ambassador apro; fi'
    fi
    (cd ${sourcedir} && module_version ${name} ) | dexec sh -c "cat > /buildroot/${name}.version && cp ${name}.version ambassador/python/"
}

summarize-sync() {
    name=$1
    shift
    lines=("$@")
    if [ "${#lines[@]}" != 0 ]; then
        dexec touch ${name}.dirty image.dirty
    fi
    for line in "${lines[@]}"; do
        if [[ $line = *.go ]]; then
            dexec touch go.dirty
            break
        fi
    done
    printf "     ${GRN}Synced ${#lines[@]} ${BLU}${name}${GRN} source files${END}\n"
    PARTIAL="yes"
    for i in {0..9}; do
        if [ "$i" = "${#lines[@]}" ]; then
            PARTIAL=""
            break
        fi
        line="${lines[$i]}"
        printf "       ${CYN}%s${END}\n" "$line"
    done
    if [ -n "${PARTIAL}" ]; then
        printf "       ${CYN}...${END}\n"
    fi
}

clean() {
    local cid
    # This command is similar to
    #
    #     builder | while read -r cid; do
    #
    # except that this command does *not* filter based on the
    # `builderbase=` and `envoybase=` labels, because we want to
    # garbage-collect old containers that were orphaned when either
    # the builderbase or the envoybase image changed.
    docker ps --quiet \
           --filter=label=builder \
           --filter=label="$BUILDER_NAME" \
    | while read -r cid; do
        printf "${GRN}Killing build container ${BLU}${cid}${END}\n"
        docker kill ${cid} > /dev/null 2>&1
        docker wait ${cid} > /dev/null 2>&1 || true
    done
    local nid
    nid=$(builder_network)
    if [ -n "${nid}" ] ; then
        printf "${GRN}Removing docker network ${BLU}${BUILDER_DOCKER_NETWORK} (${nid})${END}\n"
        # This will fail if the network has some other endpoints alive: silence any errors
        docker network rm ${nid} 2>&1 >/dev/null || true
    fi
}

find-modules () {
    find /buildroot -type d -mindepth 1 -maxdepth 1 \! -name bin | sort
}

cmd="${1:-builder}"

case "${cmd}" in
    clean)
        clean
        ;;
    clobber)
        clean
        vid=$(builder_volume)
        if [ -n "${vid}" ] ; then
            printf "${GRN}Killing cache volume ${BLU}${vid}${END}\n"
            if ! docker volume rm ${vid} > /dev/null 2>&1 ; then \
                printf "${RED}Could not kill cache volume; are other builders still running?${END}\n"
            fi
        fi
        ;;
    bootstrap)
        bootstrap >&2
        echo $(builder)
        ;;
    builder)
        echo $(builder)
        ;;
    sync)
        shift
        sync $1 $2 $(builder)
        ;;
    release-type)
        shift
        RELVER="$1"
        if [ -z "${RELVER}" ]; then
            source <(module_version ${BUILDER_NAME})
            RELVER="${RELEASE_VERSION}"
        fi

        if [[ "${RELVER}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
            echo release
        elif [[ "${RELVER}" =~ ^[0-9]+\.[0-9]+\.[0-9]+-rc\.[0-9]*$ ]]; then
            echo rc
        else
            echo other
        fi
        ;;
    release-version)
        shift
        eval $(module_version ${BUILDER_NAME})
        echo "${RELEASE_VERSION}"
        ;;
    is-dirty)
        shift
        eval $(module_version ${BUILDER_NAME})
        echo "${GIT_DIRTY}"
        ;;
    raw-version)
        shift
        module_version ${BUILDER_NAME}
        ;;
    version)
        shift
        eval $(module_version ${BUILDER_NAME})
        echo "${BUILD_VERSION}"
        ;;
    compile)
        shift
        dexec /buildroot/builder.sh compile-internal
        ;;
    compile-internal)
        # This runs inside the builder image
        if [[ $(find-modules) != /buildroot/ambassador* ]]; then
            echo "Error: ambassador must be the first module to build things correctly"
            echo "Modules are: $(find-modules)"
            exit 1
        fi

        for MODDIR in $(find-modules); do
            module=$(basename ${MODDIR})
            eval "$(grep BUILD_VERSION apro.version 2>/dev/null)" # this will `eval ''` for OSS-only builds, leaving BUILD_VERSION unset; dont embed the version-number in OSS Go binaries

            if [ -e ${module}.dirty ] || ([ "$module" != ambassador ] && [ -e go.dirty ]) ; then
                if [ -e "${MODDIR}/go.mod" ]; then
                    printf "${CYN}==> ${GRN}Building ${BLU}${module}${GRN} go code${END}\n"
                    echo_on
                    mkdir -p /buildroot/bin
                    TIMEFORMAT="     (go build took %1R seconds)"
                    (cd ${MODDIR} && time go build -trimpath ${BUILD_VERSION:+ -ldflags "-X main.Version=$BUILD_VERSION" } -o /buildroot/bin ./cmd/...) || exit 1
                    TIMEFORMAT="     (${MODDIR}/post-compile took %1R seconds)"
                    if [ -e ${MODDIR}/post-compile.sh ]; then (cd ${MODDIR} && time bash post-compile.sh); fi
                    unset TIMEFORMAT
                    echo_off
                fi
            fi

            if [ -e ${module}.dirty ]; then
                if [ -e "${MODDIR}/python" ]; then
                    if ! [ -e ${MODDIR}/python/*.egg-info ]; then
                        printf "${CYN}==> ${GRN}Setting up ${BLU}${module}${GRN} python code${END}\n"
                        echo_on
                        TIMEFORMAT="     (pip install took %1R seconds)"
                        time sudo pip install --no-deps -e ${MODDIR}/python || exit 1
                        unset TIMEFORMAT
                        echo_off
                    fi
                    chmod a+x ${MODDIR}/python/*.py
                fi

                rm ${module}.dirty
            else
                printf "${CYN}==> ${GRN}Already built ${BLU}${module}${GRN}${END}\n"
            fi
        done
        rm -f go.dirty  # Do this after _all_ the Go code is built
        ;;
    mypy-internal)
        # This runs inside the builder image
        shift
        op="$1"

        # This runs inside the builder image
        if [[ $(find-modules) != /buildroot/ambassador* ]]; then
            echo "Error: ambassador must be the first module to build things correctly"
            echo "Modules are: $(find-modules)"
            exit 1
        fi

        for MODDIR in $(find-modules); do
            module=$(basename ${MODDIR})

            if [ -e "${MODDIR}/python" ]; then
                cd "${MODDIR}"

                case "$op" in
                    start)
                        if ! dmypy status >/dev/null; then
                            dmypy start -- --use-fine-grained-cache --follow-imports=skip --ignore-missing-imports
                            printf "${CYN}==> ${GRN}Started mypy server for ${BLU}$module${GRN} Python code${END}\n"
                        else
                            printf "${CYN}==> ${GRN}mypy server already running for ${BLU}$module${GRN} Python code${END}\n"
                        fi
                        ;;

                    stop)
                        printf "${CYN}==> ${GRN}Stopping mypy server for ${BLU}$module${GRN} Python code${END}"
                        dmypy stop
                        ;;

                    check)
                        printf "${CYN}==> ${GRN}Running mypy over ${BLU}$module${GRN} Python code${END}\n"
                        time dmypy check python
                        ;;
                esac
            fi
        done
        ;;

    pip-compile)
        build_builder_base --stage1-only
        printf "${GRN}Running pip-compile to update ${BLU}requirements.txt${END}\n"
        docker run --rm -i "$builder_base_image" sh -c 'tar xf - && pip-compile --allow-unsafe -q >&2 && cat requirements.txt' \
               < <(cd "$DIR" && tar cf - requirements.in requirements.txt) \
               > "$DIR/requirements.txt.tmp"
        mv -f "$DIR/requirements.txt.tmp" "$DIR/requirements.txt"
        ;;

    pytest-local)
        fail=""
        mkdir -p ${TEST_DATA_DIR}

        if [ -z "$SOURCE_ROOT" ] ; then
            export SOURCE_ROOT="$PWD"
        fi

        if [ -z "$MODDIR" ] ; then
            export MODDIR="$PWD"
        fi

        if [ -z "$ENVOY_PATH" ] ; then
            export ENVOY_PATH="${MODDIR}/bin/envoy"
        fi
        if [ ! -f "$ENVOY_PATH" ] ; then
            echo "Envoy not found at ENVOY_PATH=$ENVOY_PATH"
            exit 1
        fi

        if [ -z "$KUBESTATUS_PATH" ] ; then
            export KUBESTATUS_PATH="${MODDIR}/bin/kubestatus"
        fi
        if [ ! -f "$KUBESTATUS_PATH" ] ; then
            echo "Kubestatus not found at $KUBESTATUS_PATH"
            exit 1
        fi

        echo "$0: EDGE_STACK=$EDGE_STACK"
        echo "$0: SOURCE_ROOT=$SOURCE_ROOT"
        echo "$0: MODDIR=$MODDIR"
        echo "$0: ENVOY_PATH=$ENVOY_PATH"
        echo "$0: KUBESTATUS_PATH=$KUBESTATUS_PATH"
        if ! (cd ${MODDIR} && pytest --cov-branch --cov=ambassador --cov-report html:/tmp/cov_html --junitxml=${TEST_DATA_DIR}/pytest.xml --tb=short -rP "${pytest_args[@]}") then
            fail="yes"
        fi

        if [ "${fail}" = yes ]; then
            exit 1
        fi
        ;;

    pytest-local-unit)
        fail=""
        mkdir -p ${TEST_DATA_DIR}

        if [ -z "$SOURCE_ROOT" ] ; then
            export SOURCE_ROOT="$PWD"
        fi

        if [ -z "$MODDIR" ] ; then
            export MODDIR="$PWD"
        fi

        if [ -z "$ENVOY_PATH" ] ; then
            export ENVOY_PATH="${MODDIR}/bin/envoy"
        fi
        if [ ! -f "$ENVOY_PATH" ] ; then
            echo "Envoy not found at ENVOY_PATH=$ENVOY_PATH"
            exit 1
        fi

        echo "$0: SOURCE_ROOT=$SOURCE_ROOT"
        echo "$0: MODDIR=$MODDIR"
        echo "$0: ENVOY_PATH=$ENVOY_PATH"
        if ! (cd ${MODDIR} && pytest --cov-branch --cov=ambassador --cov-report html:/tmp/cov_html --junitxml=${TEST_DATA_DIR}/pytest.xml --tb=short -rP "${pytest_args[@]}") then
            fail="yes"
        fi

        if [ "${fail}" = yes ]; then
            exit 1
        fi
        ;;

    pytest-internal)
        # This runs inside the builder image
        fail=""
        mkdir -p ${TEST_DATA_DIR}
        for MODDIR in $(find-modules); do
            if [ -e "${MODDIR}/python" ]; then
                if ! (cd ${MODDIR} && pytest --cov-branch --cov=ambassador --cov-report html:/tmp/cov_html --junitxml=${TEST_DATA_DIR}/pytest.xml --tb=short -ra "${pytest_args[@]}") then
                   fail="yes"
                fi
            fi
        done

        if [ "${fail}" = yes ]; then
            exit 1
        fi
        ;;
    gotest-local)
        [ -n "${TEST_XML_DIR}" ] && mkdir -p ${TEST_XML_DIR}
        fail=""
        for MODDIR in ${GOTEST_MODDIRS} ; do
            if [ -e "${MODDIR}/go.mod" ]; then
                pkgs=$(cd ${MODDIR} && go list -f='{{ if or (gt (len .TestGoFiles) 0) (gt (len .XTestGoFiles) 0) }}{{ .ImportPath }}{{ end }}' ${GOTEST_PKGS})
                if [ -n "${pkgs}" ]; then
                    modname=`basename ${MODDIR}`
                    junitarg=
                    if [[ -n "${TEST_XML_DIR}" ]] ; then
                        junitarg="--junitfile ${TEST_XML_DIR}/${modname}-gotest.xml"
                    fi
                    if ! (cd ${MODDIR} && gotestsum ${junitarg} --rerun-fails=3 --format=testname --packages="${pkgs}" -- -v ${GOTEST_ARGS}) ; then
                       fail="yes"
                    fi
                fi
            fi
        done

        if [ "${fail}" = yes ]; then
            exit 1
        fi
        ;;
    build-builder-base)
        build_builder_base >&2
        echo "${builder_base_image}"
        ;;
    shell)
        echo
        docker exec -it "$(builder)" /bin/bash
        ;;
    *)
        echo "usage: builder.sh [bootstrap|builder|clean|clobber|compile|build-builder-base|shell]"
        exit 1
        ;;
esac
