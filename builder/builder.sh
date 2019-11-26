#!/usr/bin/env bash

shopt -s expand_aliases

alias echo_on="{ set -x; }"
alias echo_off="{ set +x; } 2>/dev/null"

# Choose colors carefully. If they don't work on both a black
# background and a white background, pick other colors (so white,
# yellow, and black are poor choices).
RED=$'\e[1;31m'
GRN=$'\e[1;32m'
BLU=$'\e[1;34m'
CYN=$'\e[1;36m'
END=$'\e[0m'

set -e

SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ]; do # resolve $SOURCE until the file is no longer a symlink
    DIR="$( cd -P "$( dirname "$SOURCE" )" >/dev/null 2>&1 && pwd )"
    SOURCE="$(readlink "$SOURCE")"
    [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE" # if $SOURCE was a relative symlink, we need to resolve it relative to the path where the symlink file was located
done
DIR="$( cd -P "$( dirname "$SOURCE" )" >/dev/null 2>&1 && pwd )"

DBUILD=${DIR}/dbuild.sh

# Allow the user to set a DOCKER_RUN environment variable command for
# running a container (ie, "docker run").
#
# shellcheck disable=SC2206
docker_run=(${DOCKER_RUN:-docker run})

# the name of the Doccker network
# note: use your local k3d/microk8s/kind network for running tests
DOCKER_NETWORK=${DOCKER_NETWORK:-${BUILDER_NAME}}

builder() { docker ps -q -f label=builder -f label="$BUILDER_NAME"; }
builder_network() { docker network ls -q -f name="$DOCKER_NETWORK"; }

builder_volume() { docker volume ls -q -f label=builder; }

declare -a dsynced

dsync() {
    printf "%sSynchronizing... %s%s\n" "$GRN" "$*" "$END"
    mapfile -t dsynced < <(rsync --info=name -aO -e 'docker exec -i' "$@" 2> >(grep -Fv 'rsync: failed to set permissions on' >&2))
}

dexec() {
    if [[ -t 0 ]]; then
        flags=-it
    else
        flags=-i
    fi
    docker exec ${flags} "$(builder)" "$@"
}

bootstrap() {
    if [ -z "$(builder_volume)" ] ; then
        docker volume create --label builder
        printf "%sCreated docker volume %s%s%s for caching%s\n" "$GRN" "$BLU" "$(builder_volume)" "$GRN" "$END"
    fi

    if [ -z "$(builder_network)" ]; then
        docker network create "$DOCKER_NETWORK" > /dev/null
        printf "%sCreated docker network %s%s%s\n" "$GRN" "$BLU" "$DOCKER_NETWORK" "$END"
    else
        printf "%sConnecting to existing network %s%s%s%s\n" "$GRN" "$BLU" "$DOCKER_NETWORK" "$GRN" "$END"
    fi

    if [ -z "$(builder)" ] ; then
        printf "%s==> %sBootstrapping build image%s\n" "$CYN" "$GRN" "$END"
        "$DBUILD" --target builder "$DIR" -t builder
        if [ "$(uname -s)" == Darwin ]; then
            DOCKER_GID=$(stat -f "%g" /var/run/docker.sock)
        else
            DOCKER_GID=$(stat -c "%g" /var/run/docker.sock)
        fi
        if [ -z "$DOCKER_GID" ]; then
            echo "Unable to determine docker group-id"
            exit 1
        fi

        # Allow the user to set BUILDER_MOUNTS and BUILDER_PORTMAPS
        # environment variables.
        #
        # shellcheck disable=SC2206
        local builder_mounts=($BUILDER_MOUNTS)
        # shellcheck disable=SC2206
        local builder_portmaps=($BUILDER_PORTMAPS)

        echo_on
        "${docker_run[@]}" \
            --network="$DOCKER_NETWORK" \
            --network-alias="builder" \
            --group-add="$DOCKER_GID" \
            --detach \
            --rm \
            --volume=/var/run/docker.sock:/var/run/docker.sock \
            --volume="$(builder_volume)":/home/dw \
            "${builder_mounts[@]/#/--volume=}" \
            --cap-add=NET_ADMIN \
            --label=builder \
            --label="$BUILDER_NAME" \
            "${builder_portmaps[@]/#/--publish=}" \
            --env=BUILDER_NAME="$BUILDER_NAME" \
            --entrypoint=tail builder -f /dev/null > /dev/null
        echo_off

        printf "%sStarted build container %s%s%s\n" "$GRN" "$BLU" "$(builder)" "$END"
    fi

    dsync "${DIR}/builder.sh" "$(builder)":/buildroot
    dsync "${DIR}/builder_bash_rc" "$(builder)":/home/dw/.bashrc
}

module_version() (
    shopt -s extglob
    set -o nounset
    set -o errexit

    # shellcheck disable=SC2030
    local \
        MODULE \
        GIT_BRANCH \
        GIT_COMMIT \
        GIT_DIRTY \
        GIT_DESCRIPTION \
        RELEASE_VERSION \
        BUILD_VERSION

    MODULE="$1"

    # This is only "kinda" the git branch name:
    #
    #  - if checked out is the synthetic merge-commit for a PR, then use
    #    the PR's branch name (even though the merge commit we have
    #    checked out isn't part of the branch")
    #  - if this is a CI run for a tag (not a branch or PR), then use the
    #    tag name
    #  - if none of the above, then use the actual git branch name
    #
    # read: https://graysonkoonce.com/getting-the-current-branch-name-during-a-pull-request-in-travis-ci/
    GIT_BRANCH="${TRAVIS_PULL_REQUEST_BRANCH:-${TRAVIS_BRANCH:-$(git rev-parse --abbrev-ref HEAD)}}"

    # The short git commit hash
    GIT_COMMIT="$(git rev-parse --short HEAD)"

    # Whether `git add . && git commit` would commit anything (empty=false, nonempty=true)
    if [ -n "$(git status --porcelain)" ]; then
        GIT_DIRTY='dirty'
    else
        GIT_DIRTY=''
    fi

    # The _previous_ tag, plus a git delta, like 0.36.0-436-g8b8c5d3
    GIT_DESCRIPTION="$(git describe --tags)"

    # RELEASE_VERSION is an X.Y.Z[-prerelease] (semver) string that we
    # will upload/release the image as.  It does NOT include a leading 'v'
    # (trimming the 'v' from the git tag is what the 'patsubst' is for).
    # If this is an RC or EA, then it includes the '-rcN' or '-eaN'
    # suffix.
    RELEASE_VERSION="${TRAVIS_TAG:-$(git describe --tags --always)}"
    RELEASE_VERSION="${RELEASE_VERSION#v}"
    RELEASE_VERSION+="${GIT_DIRTY:+-dirty}"

    # BUILD_VERSION is of the same format, but is the version number that
    # we build into the image.  Because an image built as a "release
    # candidate" will ideally get promoted to be the GA image, we trim off
    # the '-rcN' suffix.
    BUILD_VERSION="${RELEASE_VERSION%%-rc*([0-9])}"

    printf '%s=%q\n' \
           MODULE "$MODULE" \
           GIT_BRANCH "$GIT_BRANCH" \
           GIT_COMMIT "$GIT_COMMIT" \
           GIT_DIRTY "$GIT_DIRTY" \
           GIT_DESCRIPTION "$GIT_DESCRIPTION" \
           RELEASE_VERSION "$RELEASE_VERSION" \
           BUILD_VERSION "$BUILD_VERSION"
)

sync() {
    name=$1
    sourcedir=$2
    container=$3

    real=$(cd "$sourcedir"; pwd)

    dexec mkdir -p "/buildroot/${name}"
    dsync --exclude-from="${DIR}/sync-excludes.txt" --delete "${real}/" "${container}:/buildroot/${name}"
    summarize-sync "$name" "${dsynced[@]}"
    # shellcheck disable=SC2016
    (cd "$sourcedir" && module_version "$name" ) | dexec sh -c 'name=$1; cat > "/buildroot/${name}.version" && cp "${name}.version" ambassador/python/' -- "$name"
}

summarize-sync() {
    name=$1
    shift
    lines=("$@")
    if [ "${#lines[@]}" != 0 ]; then
        dexec touch "${name}.dirty" image.dirty
    fi
    printf "%sSynced %s %s%s%s source files%s\n" "$GRN" "${#lines[@]}" "$BLU" "$name" "$GRN" "$END"
    PARTIAL="yes"
    for i in {0..9}; do
        if [ "$i" = "${#lines[@]}" ]; then
            PARTIAL=""
            break
        fi
        line="${lines[$i]}"
        printf "  %s%s%s\n" "$CYN" "$line" "$END"
    done
    if [ -n "${PARTIAL}" ]; then
        printf "  %s...%s\n" "$CYN" "$END"
    fi
}

clean() {
    cid=$(builder)
    if [ -n "${cid}" ] ; then
        printf "%sKilling build container %s%s%s\n" "$GRN" "$BLU" "$cid" "$END"
        docker kill "$cid" > /dev/null 2>&1
        docker wait "$cid" > /dev/null 2>&1 || true
    fi
    nid=$(builder_network)
    if [ -n "${nid}" ] ; then
        printf "%sRemoving docker network %s%s (%s)%s\n" "$GRN" "$BLU" "$DOCKER_NETWORK" "$nid" "$END"
        # This will fail if the network has some other endpoints alive: silence any errors
        docker network rm "$nid" >& /dev/null || true
    fi
}

push-image() {
    LOCAL="$1"
    REMOTE="$2"

    if ! ( dexec test -e /buildroot/pushed.log && dexec fgrep -q "${REMOTE}" /buildroot/pushed.log ); then
        printf "%s==> %sPushing %s%s%s->%s%s%s\n" "$CYN" "$GRN" "$BLU" "$LOCAL" "$GRN" "$BLU" "$REMOTE" "$END"
        docker tag "$LOCAL" "$REMOTE"
        docker push "$REMOTE"
        echo "$REMOTE" | dexec sh -c "cat >> /buildroot/pushed.log"
    else
        printf "%s==> %sAlready pushed %s%s%s->%s%s%s\n" "$CYN" "$GRN" "$BLU" "$LOCAL" "$GRN" "$BLU" "$REMOTE" "$END"
    fi
}

find-modules () {
    find /buildroot -type d -mindepth 1 -maxdepth 1 \! -name bin
}

if [[ $# -eq 0 ]]; then
    set -- builder
fi
cmd="$1"
shift
case "${cmd}" in
    clean)
        clean
        ;;
    clobber)
        clean
        vid=$(builder_volume)
        if [ -n "${vid}" ] ; then
            printf "%sKilling cache volume %s%s%s\n" "$GRN" "$BLU" "$vid" "$END"
            if ! docker volume rm "$vid" > /dev/null 2>&1 ; then \
                printf "%sCould not kill cache volume; are other builders still running?%s\n" "$RED" "$END"
            fi
        fi
        ;;
    bootstrap)
        bootstrap
        builder
        ;;
    builder)
        builder
        ;;
    sync)
        bootstrap
        sync "$1" "$2" "$(builder)"
        ;;
    release-type)
        RELVER="$1"
        if [ -z "$RELVER" ]; then
            eval "$(module_version "$BUILDER_NAME")"
            # shellcheck disable=SC2031
            RELVER="$RELEASE_VERSION"
        fi

        if [[ "$RELVER" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
            echo release
        elif [[ "$RELVER" =~ ^[0-9]+\.[0-9]+\.[0-9]+-rc[0-9]*$ ]]; then
            echo rc
        else
            echo other
        fi
        ;;
    release-version)
        eval "$(module_version "$BUILDER_NAME")"
        # shellcheck disable=SC2031
        echo "$RELEASE_VERSION"
        ;;
    version)
        eval "$(module_version "$BUILDER_NAME")"
        # shellcheck disable=SC2031
        echo "$BUILD_VERSION"
        ;;
    compile)
        bootstrap
        dexec /buildroot/builder.sh compile-internal
        ;;
    compile-internal)
        # This runs inside the builder image
        for MODDIR in $(find-modules); do
            module=$(basename "$MODDIR")
            eval "$(grep BUILD_VERSION apro.version 2>/dev/null)" # this will `eval ''` for OSS-only builds, leaving BUILD_VERSION unset; don't embed the version-number in OSS Go binaries

            if [ -e "$module.dirty" ]; then
                if [ -e "${MODDIR}/go.mod" ]; then
                    printf "%s==> %sBuilding %s%s%s go code%s\n" "$CYN" "$GRN" "$BLU" "$module" "$GRN" "$END"
                    echo_on
                    mkdir -p /buildroot/bin
                    # shellcheck disable=SC2031
                    (cd "$MODDIR" && CGO_ENABLED=0 go build -trimpath ${BUILD_VERSION:+ -ldflags "-X main.Version=$BUILD_VERSION" } -o /buildroot/bin ./cmd/...) || exit 1
                    if [ -e "${MODDIR}/post-compile.sh" ]; then (cd "${MODDIR}" && bash ./post-compile.sh); fi
                    echo_off
                fi

                if [ -e "${MODDIR}/python" ]; then
                    shopt -s nullglob
                    egginfos=("$MODDIR"/python/*.egg-info)
                    if [[ ${#egginfos[@]} -eq 0 ]]; then
                        printf "%s==> %sSetting up %s%s%s python code%s\n" "$CYN" "$GRN" "$BLU" "$module" "$GRN" "$END"
                        echo_on
                        sudo pip install --no-deps -e "${MODDIR}/python" || exit 1
                        echo_off
                    fi
                    chmod a+x "${MODDIR}/python"/*.py
                fi

                rm "${module}.dirty"
            else
                printf "%s==> %sAlready built %s%s%s%s\n" "$CYN" "$GRN" "$BLU" "$module" "$GRN" "$END"
            fi
        done
        ;;
    pytest-internal)
        # This runs inside the builder image
        fail=""
        for MODDIR in $(find-modules); do
            if [ -e "${MODDIR}/python" ]; then
                # Allow the user to set a PYTEST_ARGS environment variable.
                # shellcheck disable=SC2206
                pytest_args=($PYTEST_ARGS)
                if ! (cd "$MODDIR" && pytest --tb=short -ra "${pytest_args[@]}") then
                   fail="yes"
                fi
            fi
        done

        if [ "${fail}" = yes ]; then
            exit 1
        fi
        ;;
    gotest-internal)
        # This runs inside the builder image
        fail=""
        for MODDIR in $(find-modules); do
            if [ -e "${MODDIR}/go.mod" ]; then
                # Allow the user to set a GOTEST_PKGS environment variable.
                # shellcheck disable=SC2206
                gotest_pkgs=($GOTEST_PKGS)
                mapfile -t pkgs <<<"$(cd "$MODDIR" && go list -f='{{ if or (gt (len .TestGoFiles) 0) (gt (len .XTestGoFiles) 0) }}{{ .ImportPath }}{{ end }}' "${gotest_pkgs[@]}")"

                if [ "${#pkgs[@]}" -gt 0 ]; then
                    # Allow the user to set a GOTEST_ARGS environment variable.
                    # shellcheck disable=SC2206
                    gotest_args=($GOTEST_ARGS)
                    if ! (cd "$MODDIR" && go test "${pkgs[@]}" "${gotest_args[@]}") then
                       fail="yes"
                    fi
                fi
            fi
        done

        if [ "${fail}" = yes ]; then
            exit 1
        fi
        ;;
    commit)
        name=$1
        if [ -z "${name}" ]; then
            echo "usage: ./builder.sh commit <image-name>"
            exit 1
        fi
        if dexec test -e /buildroot/image.dirty; then
            printf "%s==> %sSnapshotting %sbuilder%s image%s\n" "$CYN" "$GRN" "$BLU" "$GRN" "$END"
            docker rmi -f "$name" &> /dev/null
            docker commit -c 'ENTRYPOINT [ "/bin/bash" ]' "$(builder)" "$name"
            printf "%s==> %sBuilding %s%s%s\n" "$CYN" "$GRN" "$BLU" "$BUILDER_NAME" "$END"
            "$DBUILD" "$DIR" --build-arg artifacts="$name" --target ambassador -t "$BUILDER_NAME"
            printf "%s==> %sBuilding %skat-client%s\n" "$CYN" "$GRN" "$BLU" "$END"
            "$DBUILD" "$DIR" --build-arg artifacts="$name" --target kat-client -t kat-client
            printf "%s==> %sBuilding %skat-server%s\n" "$CYN" "$GRN" "$BLU" "$END"
            "$DBUILD" "$DIR" --build-arg artifacts="$name" --target kat-server -t kat-server
        fi
        dexec rm -f /buildroot/image.dirty
        ;;
    push)
        push-image "$BUILDER_NAME" "$1"
        push-image kat-client "$2"
        push-image kat-server "$3"
        ;;
    shell)
        bootstrap
        printf "\n"
        docker exec -it "$(builder)" /bin/bash
        ;;
    *)
        echo "usage: builder.sh [bootstrap|builder|clean|clobber|compile|commit|shell]"
        exit 1
        ;;
esac
