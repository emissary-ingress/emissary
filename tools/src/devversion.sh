#!/bin/bash

set -euE -o pipefail

# shellcheck disable=SC2016
print_usage() {
    printf 'Usage: %s [--help|DOCKER_REPO]\n' "$0"
    printf 'Determine the $(tools/goversion)-string that HEAD was built with in CI\n'
    echo
    printf 'This works by calling `$(tools/goversion) --all`, and checking which of\n'
    printf 'those can be pulled from DOCKER_REPO.  If DOCKER_REPO is not given as an\n'
    printf 'argument, then `${DEV_REGISTRY}/emissary` is used.\n'
    echo
    printf 'This may be slow because it fully pulls the Docker image, rather than just\n'
    printf 'querying for existance.  This is in part because gcr.io doesn'\''t allow\n'
    printf '`docker manifest inspect`, but also the only time you'\''d want this version\n'
    printf 'number is if you'\''re about to pull the image anyway.\n'
    echo
    printf 'If none of the tags work, then it will back off and try again until it hits a\n'
    printf 'timeout; under the assumption that it is running concurrently with CI the job\n'
    printf 'that pushes the image, and that it should just wait for that other job to\n'
    printf 'finish.\n'
}

errusage() {
	printf >&2 '%s: error: %s\n' "$0" "$*"
	printf >&2 "Try '%s --help' for more information\n" "$0"
	exit 2
}

msg() {
    local str
    # shellcheck disable=SC2059
    printf -v str "$@"
    printf >&2 '[%s] %s\n' "${0##*/}" "$str"
}

main() {
    local docker_repo
    case $# in
        0)
            if [[ -z "${DEV_REGISTRY:-}" ]]; then
                errusage "must either provide a Docker repo or set DEV_REGISTRY"
            fi
            docker_repo="${DEV_REGISTRY}/emissary"
            ;;
        1)
            if [[ "$1" == '--help' ]]; then
                print_usage
                return 0
            fi
            docker_repo="$1"
            ;;
        *)
            errusage "expected 0 or 1 arguments, got $#"
            ;;
    esac

    msg 'docker_repo=%q' "$docker_repo"

    # These tunables currently mimic the old
    # releng/release-wait-for-commit tool's values.
    timeout_secs=600
    backoff_secs=30

    start_time=$(date +%s)
    deadline=$(( start_time + timeout_secs ))
    while (( $(date +%s) < deadline )); do
        local vsemver
        while read -r vsemver; do
            msg 'checking %q...' "$vsemver"
            if docker pull "${docker_repo}:${vsemver#v}" &>/dev/null; then
                msg 'found %q!' "$vsemver"
                printf '%s\n' "$vsemver"
                return 0
            fi
        done < <("${0%/*}"/goversion --all | grep '^v2\.')
        msg 'backing off for %ds then retrying...' "$backoff_secs"
        sleep "$backoff_secs"
    done

    msg 'unable to find a Docker image in %q that matches HEAD', "$docker_repo"
    return 1
}

main "$@"
