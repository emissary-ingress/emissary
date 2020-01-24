#!/usr/bin/env bash

shopt -s expand_aliases

alias echo_on="{ set -x; }"
alias echo_off="{ set +x; } 2>/dev/null"

# Choose colors carefully. If they don't work on both a black
# background and a white background, pick other colors (so white,
# yellow, and black are poor choices).
RED='\033[1;31m'
GRN='\033[1;32m'
BLU='\033[1;34m'
CYN='\033[1;36m'
END='\033[0m'

set -e

SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ]; do # resolve $SOURCE until the file is no longer a symlink
    DIR="$( cd -P "$( dirname "$SOURCE" )" >/dev/null 2>&1 && pwd )"
    SOURCE="$(readlink "$SOURCE")"
    [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE" # if $SOURCE was a relative symlink, we need to resolve it relative to the path where the symlink file was located
done
DIR="$( cd -P "$( dirname "$SOURCE" )" >/dev/null 2>&1 && pwd )"

DBUILD=${DIR}/dbuild.sh

# command for running a container (ie, "docker run")
DOCKER_RUN=${DOCKER_RUN:-docker run}

# the name of the Doccker network
# note: use your local k3d/microk8s/kind network for running tests
DOCKER_NETWORK=${DOCKER_NETWORK:-${BUILDER_NAME}}

builder() { docker ps -q -f label=builder -f label="${BUILDER_NAME}"; }
builder_network() { docker network ls -q -f name="${DOCKER_NETWORK}"; }

builder_volume() { docker volume ls -q -f label=builder; }

declare -a dsynced

dsync() {
    printf "${GRN}Synchronizing... $*${END}\n"
    IFS='|' read -ra dsynced <<<"$(rsync --info=name -aO -e 'docker exec -i' $@ 2> >(fgrep -v 'rsync: failed to set permissions on' >&2) | tr '\n' '|')"
}

dexec() {
    if [[ -t 0 ]]; then
        flags=-it
    else
        flags=-i
    fi
    docker exec ${flags} $(builder) "$@"
}

bootstrap() {
    if [ -z "$(builder_volume)" ] ; then
        docker volume create --label builder
        printf "${GRN}Created docker volume ${BLU}$(builder_volume)${GRN} for caching${END}\n"
    fi

    if [ -z "$(builder_network)" ]; then
        docker network create "${DOCKER_NETWORK}" > /dev/null
        printf "${GRN}Created docker network ${BLU}${DOCKER_NETWORK}${END}\n"
    else
        printf "${GRN}Connecting to existing network ${BLU}${DOCKER_NETWORK}${GRN}${END}\n"
    fi

    if [ -z "$(builder)" ] ; then
        printf "${CYN}==> ${GRN}Bootstrapping build image${END}\n"
        ${DBUILD} --target builder ${DIR} -t builder
        if [ "$(uname -s)" == Darwin ]; then
            DOCKER_GID=$(stat -f "%g" /var/run/docker.sock)
        else
            DOCKER_GID=$(stat -c "%g" /var/run/docker.sock)
        fi
        if [ -z "${DOCKER_GID}" ]; then
            echo "Unable to determine docker group-id"
            exit 1
        fi

        echo_on
        $DOCKER_RUN --network "${DOCKER_NETWORK}" --network-alias "builder" --group-add ${DOCKER_GID} -d --rm -v /var/run/docker.sock:/var/run/docker.sock -v $(builder_volume):/home/dw ${BUILDER_MOUNTS} --cap-add NET_ADMIN -lbuilder -l${BUILDER_NAME} ${BUILDER_PORTMAPS} -e BUILDER_NAME=${BUILDER_NAME} --entrypoint tail builder -f /dev/null > /dev/null
        echo_off

        printf "${GRN}Started build container ${BLU}$(builder)${END}\n"
    fi

    dsync ${DIR}/builder.sh $(builder):/buildroot
    dsync ${DIR}/builder_bash_rc $(builder):/home/dw/.bashrc
}

module_version() {
    echo MODULE="\"$1\""
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
    for VAR in "${TRAVIS_PULL_REQUEST_BRANCH}" "${TRAVIS_BRANCH}" $(git rev-parse --abbrev-ref HEAD); do
        if [ -n "${VAR}" ]; then
            echo GIT_BRANCH="\"${VAR}\""
            break
        fi
    done
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
    # The _previous_ tag, plus a git delta, like 0.36.0-436-g8b8c5d3
    echo GIT_DESCRIPTION="\"$(git describe --tags)\""

    # RELEASE_VERSION is an X.Y.Z[-prerelease] (semver) string that we
    # will upload/release the image as.  It does NOT include a leading 'v'
    # (trimming the 'v' from the git tag is what the 'patsubst' is for).
    # If this is an RC or EA, then it includes the '-rc.N' or '-ea.N'
    # suffix.
    #
    # BUILD_VERSION is of the same format, but is the version number that
    # we build into the image.  Because an image built as a "release
    # candidate" will ideally get promoted to be the GA image, we trim off
    # the '-rcN' suffix.
    for VAR in "${TRAVIS_TAG}" "$(git describe --tags --always)"; do
        if [ -n "${VAR}" ]; then
            RELEASE_VERSION="${VAR}"
            break
        fi
    done

    if [[ ${RELEASE_VERSION} =~ ^v[0-9]+.*$ ]]; then
        RELEASE_VERSION=${RELEASE_VERSION:1}
    fi

    if [ -n "${dirty}" ]; then
        RELEASE_VERSION="${RELEASE_VERSION}-dirty"
    fi

    echo RELEASE_VERSION="\"${RELEASE_VERSION}\""
    echo BUILD_VERSION="\"$(echo "${RELEASE_VERSION}" | sed 's/-rc\.[0-9]*$//')\""
}

sync() {
    name=$1
    sourcedir=$2
    container=$3

    real=$(cd ${sourcedir}; pwd)

    dexec mkdir -p /buildroot/${name}
    dsync --exclude-from=${DIR}/sync-excludes.txt --delete ${real}/ ${container}:/buildroot/${name}
    summarize-sync $name "${dsynced[@]}"
    (cd ${sourcedir} && module_version ${name} ) | dexec sh -c "cat > /buildroot/${name}.version && cp ${name}.version ambassador/python/"
}

summarize-sync() {
    name=$1
    shift
    lines=("$@")
    if [ "${#lines[@]}" != 0 ]; then
        dexec touch ${name}.dirty image.dirty
    fi
    printf "${GRN}Synced ${#lines[@]} ${BLU}${name}${GRN} source files${END}\n"
    PARTIAL="yes"
    for i in {0..9}; do
        if [ "$i" = "${#lines[@]}" ]; then
            PARTIAL=""
            break
        fi
        line="${lines[$i]}"
        printf "  ${CYN}${line}${END}\n"
    done
    if [ -n "${PARTIAL}" ]; then
        printf "  ${CYN}...${END}\n"
    fi
}

clean() {
    cid=$(builder)
    if [ -n "${cid}" ] ; then
        printf "${GRN}Killing build container ${BLU}${cid}${END}\n"
        docker kill ${cid} > /dev/null 2>&1
        docker wait ${cid} > /dev/null 2>&1 || true
    fi
    nid=$(builder_network)
    if [ -n "${nid}" ] ; then
        printf "${GRN}Removing docker network ${BLU}${DOCKER_NETWORK} (${nid})${END}\n"
        # This will fail if the network has some other endpoints alive: silence any errors
        docker network rm ${nid} 2>&1 >/dev/null || true
    fi
}

push-image() {
    LOCAL="$1"
    REMOTE="$2"

    if ! ( dexec test -e /buildroot/pushed.log && dexec fgrep -q "${REMOTE}" /buildroot/pushed.log ); then
        printf "${CYN}==> ${GRN}Pushing ${BLU}${LOCAL}${GRN}->${BLU}${REMOTE}${END}\n"
        docker tag ${LOCAL} ${REMOTE}
        docker push ${REMOTE}
        echo ${REMOTE} | dexec sh -c "cat >> /buildroot/pushed.log"
    else
        printf "${CYN}==> ${GRN}Already pushed ${BLU}${LOCAL}${GRN}->${BLU}${REMOTE}${END}\n"
    fi
}

find-modules () {
    find /buildroot -type d -mindepth 1 -maxdepth 1 \! -name bin
}

cmd="$1"

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
        bootstrap
        echo $(builder)
        ;;
    builder|"")
        echo $(builder)
        ;;
    sync)
        shift
        bootstrap
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
        elif [[ "${RELVER}" =~ ^[0-9]+\.[0-9]+\.[0-9]+-rc[0-9]*$ ]]; then
            echo rc
        else
            echo other
        fi
        ;;
    release-version)
        shift
        source <(module_version ${BUILDER_NAME})
        echo "${RELEASE_VERSION}"
        ;;
    version)
        shift
        source <(module_version ${BUILDER_NAME})
        echo "${BUILD_VERSION}"
        ;;
    compile)
        shift
        bootstrap
        dexec /buildroot/builder.sh compile-internal
        ;;
    compile-internal)
        # This runs inside the builder image
        for MODDIR in $(find-modules); do
            module=$(basename ${MODDIR})
            eval "$(grep BUILD_VERSION apro.version 2>/dev/null)" # this will `eval ''` for OSS-only builds, leaving BUILD_VERSION unset; dont embed the version-number in OSS Go binaries

            if [ -e ${module}.dirty ]; then
                if [ -e "${MODDIR}/go.mod" ]; then
                    printf "${CYN}==> ${GRN}Building ${BLU}${module}${GRN} go code${END}\n"
                    echo_on
                    mkdir -p /buildroot/bin
                    (cd ${MODDIR} && go build -trimpath ${BUILD_VERSION:+ -ldflags "-X main.Version=$BUILD_VERSION" } -o /buildroot/bin ./cmd/...) || exit 1
                    if [ -e ${MODDIR}/post-compile.sh ]; then (cd ${MODDIR} && bash post-compile.sh); fi
                    echo_off
                fi

                if [ -e "${MODDIR}/python" ]; then
                    if ! [ -e ${MODDIR}/python/*.egg-info ]; then
                        printf "${CYN}==> ${GRN}Setting up ${BLU}${module}${GRN} python code${END}\n"
                        echo_on
                        sudo pip install --no-deps -e ${MODDIR}/python || exit 1
                        echo_off
                    fi
                    chmod a+x ${MODDIR}/python/*.py
                fi

                rm ${module}.dirty
            else
                printf "${CYN}==> ${GRN}Already built ${BLU}${module}${GRN}${END}\n"
            fi
        done
        ;;
    pytest-internal)
        # This runs inside the builder image
        fail=""
        for MODDIR in $(find-modules); do
            if [ -e "${MODDIR}/python" ]; then
                if ! (cd ${MODDIR} && pytest --tb=short -ra ${PYTEST_ARGS}) then
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
                pkgs=$(cd ${MODDIR} && go list -f='{{ if or (gt (len .TestGoFiles) 0) (gt (len .XTestGoFiles) 0) }}{{ .ImportPath }}{{ end }}' ${GOTEST_PKGS})

                if [ -n "${pkgs}" ]; then
                    if ! (cd ${MODDIR} && go test ${pkgs} ${GOTEST_ARGS}) then
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
        shift
        name=$1
        if [ -z "${name}" ]; then
            echo "usage: ./builder.sh commit <image-name>"
            exit 1
        fi
        if dexec test -e /buildroot/image.dirty; then
            printf "${CYN}==> ${GRN}Snapshotting ${BLU}builder${GRN} image${END}\n"
            docker rmi -f "${name}" &> /dev/null
            docker commit -c 'ENTRYPOINT [ "/bin/bash" ]' $(builder) "${name}"
            printf "${CYN}==> ${GRN}Building ${BLU}${BUILDER_NAME}${END}\n"
            ${DBUILD} ${DIR} --build-arg artifacts=${name} --target ambassador -t ${BUILDER_NAME}
            printf "${CYN}==> ${GRN}Building ${BLU}kat-client${END}\n"
            ${DBUILD} ${DIR} --build-arg artifacts=${name} --target kat-client -t kat-client
            printf "${CYN}==> ${GRN}Building ${BLU}kat-server${END}\n"
            ${DBUILD} ${DIR} --build-arg artifacts=${name} --target kat-server -t kat-server
        fi
        dexec rm -f /buildroot/image.dirty
        ;;
    push)
        shift
        push-image ${BUILDER_NAME} "$1"
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
