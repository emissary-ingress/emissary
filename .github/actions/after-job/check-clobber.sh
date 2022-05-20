#!/usr/bin/env bash
#shellcheck disable=SC2016

r=0
if [[ -n "$(git clean --dry-run -d -x)" ]]; then
    echo
    echo 'There are files that `make clobber` did not remove that it should have:'
    git clean --dry-run -d -x | sed 's/^Would remove /    /'
    echo
    r=1
fi
if docker image list --format='{{ .Repository }}:{{ .Tag }}' | grep -q '\.local/'; then
    echo
    echo 'There are Docker images that `make clobber` did not remove that it should have:'
    docker image list | grep '\.local/'
    echo
    r=1
fi
if [[ -n "$(docker container list --all --quiet)" ]]; then
    echo
    echo 'There are Docker containers that `make clobber` did not remove:'
    docker container list --all
    echo
    r=1
fi
if [[ -n "$(docker volume list --quiet)" ]]; then
    echo
    echo 'There are Docker volumes that `make clobber` did not remove:'
    docker volume list
    echo
    r=1
fi
exit "$r"
