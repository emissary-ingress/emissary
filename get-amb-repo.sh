#!/usr/bin/env bash

# Git clone ambassador to the specified checkout
AMBASSADOR_COMMIT=$(cat ambassador.commit)

set -e
PS4=
set +x

# Ensure that GIT_DIR and GIT_WORK_TREE are unset so that `git bisect`
# and friends work properly.
unset GIT_DIR GIT_WORK_TREE

if ! [ -e ambassador ]; then
	git init ambassador
	INIT=yes
fi

if ! git -C ambassador remote get-url origin &>/dev/null; then
	set -x
	git -C ambassador remote add origin https://github.com/datawire/ambassador
	git -C ambassador remote set-url --push origin no_push
fi

{ set +x 1; } 2>/dev/null

if [ -n "${INIT}" ] || [ "$(cd ambassador && git rev-parse HEAD)" != "${AMBASSADOR_COMMIT}" ]; then
	set -x
	git -C ambassador fetch
	git -C ambassador checkout -q "${AMBASSADOR_COMMIT}"
fi
