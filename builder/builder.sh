#!/bin/bash

shopt -s expand_aliases

alias echo_on="{ set -x; }"
alias echo_off="{ set +x; } 2>/dev/null"

RED='\033[1;31m'
GRN='\033[1;32m'
YEL='\033[1;33m'
BLU='\033[1;34m'
WHT='\033[1;37m'
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

builder() { docker ps -q -f label=builder; }

builder_volume() { docker volume ls -q -f label=builder; }

bootstrap() {
    if [ -z "$(builder_volume)" ] ; then
        docker volume create --label builder
        printf "${GRN}Created docker volume ${BLU}$(builder_volume)${GRN} for caching${END}\n"
    fi

    if [ -z "$(builder)" ] ; then
        printf "${WHT}==${GRN}Bootstrapping build image${WHT}==${END}\n"
        ${DBUILD} --build-arg envoy=$(cat ${DIR}/../base-envoy.docker) --target builder ${DIR} -t builder
        if [ "$(uname -s)" == Darwin ]; then
            DOCKER_GID=$(stat -f "%g" /var/run/docker.sock)
        else
            DOCKER_GID=$(stat -c "%g" /var/run/docker.sock)
        fi
        if [ -z "${DOCKER_GID}" ]; then
            echo "Unable to determine docker group-id"
            exit 1
        fi
        docker run --group-add ${DOCKER_GID} -d --rm -v /var/run/docker.sock:/var/run/docker.sock -v $(builder_volume):/home/dw --net=host --cap-add NET_ADMIN -lbuilder --entrypoint tail builder -f /dev/null > /dev/null
        printf "${GRN}Started build container ${BLU}$(builder)${END}\n"
    fi

    rsync -q -a -e 'docker exec -i' ${DIR}/builder.sh $(builder):/buildroot
}

sync() {
    name=$1
    sourcedir=$2
    container=$3

    real=$(cd ${sourcedir}; pwd)

    docker exec -i ${container} mkdir -p /buildroot/${name}
    summarize-sync $name $container $(rsync --exclude-from=${DIR}/sync-excludes.txt --info=name -a --delete -e 'docker exec -i' ${real}/ ${container}:/buildroot/${name})
}

dirty() {
    cid=$1
    docker exec -i ${cid} sh -c 'test -n "$(ls /buildroot | fgrep .dirty)"'
}

clear-dirty() {
    cid=$1
    docker exec -i ${cid} sh -c 'rm -f /buildroot/*.dirty'
}

summarize-sync() {
    name=$1
    shift
    container=$1
    shift
    if [ "$#" != 0 ]; then
        docker exec -i ${container} touch /buildroot/${name}.dirty
    fi
    printf "${GRN}Synced $# ${BLU}${name}${GRN} source files${END}\n"
    prevdel=""
    for var in "$@"; do
        if [ -n "$prevdel" ]; then
            printf "  ${YEL}deleted${END} $var\n"
        fi
        if [ "${var}" == "deleting" ]; then
            prevdel="x"
        else
            prevdel=""
        fi
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
            docker volume rm ${vid} > /dev/null 2>&1
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
    compile)
        shift
        bootstrap
        docker exec -it $(builder) /buildroot/builder.sh compile-internal
        ;;
    compile-internal)
        # This runs inside the builder image
        for SRCDIR in $(find /buildroot -type f -name go.mod -or -name python -type d -mindepth 2 -maxdepth 2); do
            module=$(basename $(dirname ${SRCDIR}))
            if [ -e ${module}.dirty ]; then
                case ${SRCDIR} in
                    */go.mod)
                        printf "${WHT}==${GRN}Building ${BLU}${module}${GRN} go code${WHT}==${END}\n"
                        wd=$(dirname ${SRCDIR})
                        echo_on
                        (cd ${wd} && GOBIN=/buildroot/bin go install ./cmd/...) || exit 1
                        echo_off
                        ;;
                    */python)
                        if ! [ -e ${SRCDIR}/*.egg-info ]; then
                            printf "${WHT}==${GRN}Setting up ${BLU}${module}${GRN} python code${WHT}==${END}\n"
                            echo_on
                            sudo pip install --no-deps -e ${SRCDIR} || exit 1
                            echo_off
                        fi
                        chmod a+x ${SRCDIR}/*.py
                        ;;
                esac
            fi
        done
        ;;
    test-internal)
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
        if [ "${fail}" == yes ]; then
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
        if dirty ${cid}; then
	    printf "${WHT}==${GRN}Snapshotting ${BLU}builder${GRN} image${WHT}==${END}\n"
	    docker rmi -f "${name}" &> /dev/null
            docker commit -c 'ENTRYPOINT [ "/bin/bash" ]' ${cid} "${name}"
        fi
        clear-dirty ${cid}
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
