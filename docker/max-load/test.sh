#!/bin/sh

preambassador_cleanup() {
	kubectl delete pod prometheus-prometheus-0
	kubectl delete daemonset ambassador || true
	kubectl delete deployment ambassador || true
	kubectl delete deployment ambassador-pro-redis || true
	kubectl delete services ambassador ambassador-pro ambassador-pro-redis || true
}

adjust_ambassador() {
	for assignment in "$@"; do
		export "$assignment"
	done

	kubeapply -f /opt/03-ambassador.yaml
	# Clean up left-overs
	if test "$USE_PRO_RATELIMIT" != true; then
		kubectl delete service ambassador-pro-redis || true
		kubectl delete deployment ambassador-pro-redis || true
	fi
	# Make sure that there are no old pods around to accidentally
	# hit, because
	# https://github.com/datawire/teleproxy/issues/103 and
	# https://github.com/datawire/teleproxy/issues/65.  This is a
	# little racy; it might delete new pods.  But that's OK; it's
	# just extra work for the cluster to re-create them.
	kubectl delete pods -l service=ambassador
}

i=0
run_test() {
	name=$1
	url=$2
	shift 2
	# Make sure that Ambassador is ready
	while ! curl --fail -Lk -is --oauth2-bearer eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ. "$url"; do
		sleep 1
	done
	# Make it easy to tell apart clusters in the graphs
	sleep 30
	# Run the test
	max-load --load-max-rps=3000 --csv-file="$((i++))-${name}.csv" "$@" "$url"
}

cd /tmp
trap 'python3 -m http.server' EXIT
set -ex

preambassador_cleanup
run_test backend-http1               http://load-http-echo/load-testing/base/ --enable-http2=false
adjust_ambassador USE_TLS=false USE_NOOP_AUTH='' USE_PRO_RATELIMIT=false USE_PRO_AUTH=false
run_test oss-http1                   http://ambassador/load-testing/base/     --enable-http2=false
adjust_ambassador USE_TLS=true
run_test oss-https1                  https://ambassador/load-testing/base/    --enable-http2=false
run_test oss-https2                  https://ambassador/load-testing/base/
adjust_ambassador USE_NOOP_AUTH=http
run_test oss-https2-httpauth         https://ambassador/load-testing/base/
adjust_ambassador USE_NOOP_AUTH=grpc
run_test oss-https2-grpcauth         https://ambassador/load-testing/base/
adjust_ambassador USE_NOOP_AUTH='' USE_PRO_RATELIMIT=true
run_test pro-rlonly-https2-rl-minute https://ambassador/load-testing/rl-minute/
run_test pro-rlonly-https2-rl-second https://ambassador/load-testing/rl-second/
adjust_ambassador USE_NOOP_AUTH='' USE_PRO_AUTH=true
run_test pro-https2-base             https://ambassador/load-testing/base/
run_test pro-https2-rl-minute        https://ambassador/load-testing/rl-minute/
run_test pro-https2-rl-second        https://ambassador/load-testing/rl-second/
run_test pro-https2-filter-jwt       https://ambassador/load-testing/filter-jwt/
