#!/bin/bash

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

builder() { docker ps -q -f label=builder -f label=${BUILDER_NAME}; }

builder_volume() { docker volume ls -q -f label=builder; }

bootstrap() {
    if [ -z "$(builder_volume)" ] ; then
        docker volume create --label builder
        printf "${GRN}Created docker volume ${BLU}$(builder_volume)${GRN} for caching${END}\n"
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
        docker run --group-add ${DOCKER_GID} -d --rm -v /var/run/docker.sock:/var/run/docker.sock -v $(builder_volume):/home/dw --net=host --cap-add NET_ADMIN -lbuilder -l${BUILDER_NAME} --entrypoint tail builder -f /dev/null > /dev/null
        printf "${GRN}Started build container ${BLU}$(builder)${END}\n"
    fi

    rsync -q -a -e 'docker exec -i' ${DIR}/builder.sh $(builder):/buildroot
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
    # If this is an RC or EA, then it includes the '-rcN' or '-eaN'
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
    echo BUILD_VERSION="\"$(echo "${RELEASE_VERSION}" | sed 's/-rc[0-9]*$//')\""
}

sync() {
    name=$1
    sourcedir=$2
    container=$3

    real=$(cd ${sourcedir}; pwd)

    docker exec -i ${container} mkdir -p /buildroot/${name}
    declare -a lines
    IFS='|' read -ra lines <<<"$(rsync --exclude-from=${DIR}/sync-excludes.txt --info=name -aO --delete -e 'docker exec -i' ${real}/ ${container}:/buildroot/${name} | tr '\n' '|')"
    summarize-sync $name $container "${lines[@]}"
    (cd ${sourcedir} && module_version ${name} ) | docker exec -i ${container} sh -c "cat > /buildroot/${name}.version && if [ -e ${name}/python ]; then cp ${name}.version ${name}/python/; fi"
}

image-dirty() {
    cid=$1
    docker exec -i ${cid} sh -c 'test -e /buildroot/image.dirty'
}

image-clear() {
    cid=$1
    docker exec -i ${cid} sh -c 'rm -f /buildroot/image.dirty'
}

summarize-sync() {
    name=$1
    shift
    container=$1
    shift
    lines=("$@")
    if [ "${#lines[@]}" != 0 ]; then
        docker exec -i ${container} touch ${name}.dirty image.dirty
    fi
    printf "${GRN}Synced ${#lines[@]} ${BLU}${name}${GRN} source files${END}\n"
    for i in {0..9}; do
        if [ "$i" = "${#lines[@]}" ]; then
            break
        fi
        line="${lines[$i]}"
        printf "  ${CYN}${line}${END}\n"
    done
}

clean() {
    cid=$(builder)
    if [ -n "${cid}" ] ; then
        printf "${GRN}Killing build container ${BLU}${cid}${END}\n"
        docker kill ${cid} > /dev/null 2>&1
        docker wait ${cid} > /dev/null 2>&1 || true
    fi
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
            docker volume rm ${vid} > /dev/null 2>&1 || true
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
        if [[ -t 0 ]]; then
            docker exec -it $(builder) /buildroot/builder.sh compile-internal
        else
            docker exec $(builder) /buildroot/builder.sh compile-internal
        fi
        ;;
    compile-internal)
        # This runs inside the builder image
        for SRCDIR in $(find /buildroot -type f -name go.mod -or -name python -type d -mindepth 2 -maxdepth 2); do
            module=$(basename $(dirname ${SRCDIR}))
            if [ -e ${module}.dirty ]; then
                case ${SRCDIR} in
                    */go.mod)
                        printf "${CYN}==> ${GRN}Building ${BLU}${module}${GRN} go code${END}\n"
                        wd=$(dirname ${SRCDIR})
                        echo_on
                        (cd ${wd} && GOBIN=/buildroot/bin go install $(if [[ -f vendor/modules.txt ]]; then echo ' -mod=vendor '; fi) ./cmd/...) || exit 1
                        if [ -e ${wd}/post-compile.sh ]; then (cd ${wd} && bash post-compile.sh); fi
                        echo_off
                        ;;
                    */python)
                        if ! [ -e ${SRCDIR}/*.egg-info ]; then
                            printf "${CYN}==> ${GRN}Setting up ${BLU}${module}${GRN} python code${END}\n"
                            echo_on
                            sudo pip install --no-deps -e ${SRCDIR} || exit 1
                            echo_off
                        fi
                        chmod a+x ${SRCDIR}/*.py
                        ;;
                esac
                rm ${module}.dirty
            fi
        done
        ;;
    pytest-internal)
        # This runs inside the builder image
        fail=""
        for SRCDIR in $(find /buildroot -type d -name python -mindepth 2 -maxdepth 2); do
            module=$(basename $(dirname ${SRCDIR}))
            wd=$(dirname ${SRCDIR})
            if ! (cd ${wd} && pytest --tb=short -ra ${PYTEST_ARGS}) then
               fail="yes"
            fi
        done
        if [ "${fail}" = yes ]; then
            exit 1
        fi
        ;;
    gotest-internal)
        # This runs inside the builder image
        fail=""
        for SRCDIR in $(find /buildroot -type f -name go.mod -mindepth 2 -maxdepth 2); do
            module=$(basename $(dirname ${SRCDIR}))
            wd=$(dirname ${SRCDIR})
            pkgs=$(cd ${wd} && go list -f='{{ if or (gt (len .TestGoFiles) 0) (gt (len .XTestGoFiles) 0) }}{{ .ImportPath }}{{ end }}' ${GOTEST_PKGS})
            if [ -n "${pkgs}" ]; then
                if ! (cd ${wd} && go test ${pkgs} ${GOTEST_ARGS}) then
                   fail="yes"
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
        cid=$(builder)
        if image-dirty ${cid}; then
	    printf "${CYN}==> ${GRN}Snapshotting ${BLU}builder${GRN} image${END}\n"
	    docker rmi -f "${name}" &> /dev/null
            docker commit -c 'ENTRYPOINT [ "/bin/bash" ]' ${cid} "${name}"
	    printf "${CYN}==> ${GRN}Building ${BLU}${BUILDER_NAME}${GRN} image${END}\n"
	    ${DBUILD} ${DIR} --build-arg artifacts=${name} --target ambassador -t ${BUILDER_NAME}
	    printf "${CYN}==> ${GRN}Building ${BLU}kat-client${GRN} image${END}\n"
	    ${DBUILD} ${DIR} --build-arg artifacts=${name} --target kat-client -t kat-client
	    printf "${CYN}==> ${GRN}Building ${BLU}kat-server${GRN} image${END}\n"
	    ${DBUILD} ${DIR} --build-arg artifacts=${name} --target kat-server -t kat-server
        fi
        image-clear ${cid}
        ;;
    push)
        shift
        AMB_IMAGE=$1
        KAT_CLI_IMAGE=$2
        KAT_SRV_IMAGE=$3

        cid=$(builder)

        if ! docker exec -i ${cid} fgrep -q "${AMB_IMAGE}" /buildroot/pushed.log; then
	    printf "${CYN}==> ${GRN}Pushing ${BLU}${BUILDER_NAME}${GRN} image${END}\n"
	    docker tag ${BUILDER_NAME} ${AMB_IMAGE}
	    docker push ${AMB_IMAGE}
	    echo ${AMB_IMAGE} | docker exec -i ${cid} sh -c "cat >> /buildroot/pushed.log"
        fi

        if ! docker exec -i ${cid} fgrep -q "${KAT_CLI_IMAGE}" /buildroot/pushed.log; then
	    printf "${CYN}==> ${GRN}Pushing ${BLU}kat-client${GRN} image${END}\n"
	    docker tag kat-client ${KAT_CLI_IMAGE}
	    docker push ${KAT_CLI_IMAGE}
	    echo ${KAT_CLI_IMAGE} | docker exec -i ${cid} sh -c "cat >> /buildroot/pushed.log"
        fi

        if ! docker exec -i ${cid} fgrep -q "${KAT_SRV_IMAGE}" /buildroot/pushed.log; then
	    printf "${CYN}==> ${GRN}Pushing ${BLU}kat-server${GRN} image${END}\n"
	    docker tag kat-server ${KAT_SRV_IMAGE}
	    docker push ${KAT_SRV_IMAGE}
	    echo ${KAT_SRV_IMAGE} | docker exec -i ${cid} sh -c "cat >> /buildroot/pushed.log"
        fi
        ;;
    shell)
        bootstrap
        docker exec -it "$(builder)" /bin/bash
        ;;
    *)
        echo "usage: builder.sh [bootstrap|builder|clean|clobber|compile|commit|shell]"
        exit 1
        ;;
esac
