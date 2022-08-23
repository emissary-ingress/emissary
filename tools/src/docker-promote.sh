#!/usr/bin/env bash

set -euE

usage() {
	echo "Usage: ${0##*/} FROM_IMAGE TO_REPO:TO_VERSION"
}

errUsage() {
	printf '%s: error: %s' "$*"
	usage >&2
	exit 2
}

main() {
	for arg in "$@"; do
		if [[ $arg == '--help' ]]; then
			usage
			exit 0
		fi
	done
	if [[ $# != 2 ]]; then
		errUsage "expected exactly 2 arguments, got $#"
	fi
	from=$1
	to=$2
	if [[ $to != *:* ]]; then
		errUsage "does not look like a REPO:VERSION pair: ${to}"
	fi
	toVersion=${to##*:}
	if [[ $toVersion == v* ]]; then
		errUsage "Docker tags should not have a 'v' suffix: ${to}"
	fi

	tmpdir=$(mktemp -d -t docker-promote.XXXXXXXXXX)
	trap 'rm -rf "$tmpdir"' EXIT
	cat >"$tmpdir/Dockerfile" <<-EOF
		FROM ${from}
		RUN find / -name ambassador.version -exec sed -i \\
		    -e '1s/.*/${toVersion}/' \\
		    -- {} +
	EOF

	docker build -t "$to" "$tmpdir"
}

main "$@"
