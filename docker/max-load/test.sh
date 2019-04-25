#!/bin/sh

preambassador_cleanup() {
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
	if test "$USE_PRO_RATELIMIT" != true; then
		kubectl delete service ambassador-pro-redis || true
		kubectl delete deployment ambassador-pro-redis || true
	fi
	sleep 5
}

i=0
run_test() {
	max-load --load-max-rps=300 --csv-file=$((i++))-"$@"
	sleep 30
}

cd /tmp
trap 'python3 -m http.server' EXIT
set -ex

preambassador_cleanup
run_test backend-http1.csv         --enable-http2=false  http://load-http-echo/load-testing/base/
adjust_ambassador USE_TLS=false USE_NOOP_AUTH='' USE_PRO_RATELIMIT=false USE_PRO_AUTH=false
run_test oss-http1.csv             --enable-http2=false  http://ambassador/load-testing/base/
adjust_ambassador USE_TLS=true
run_test oss-https1.csv            --enable-http2=false https://ambassador/load-testing/base/
run_test oss-https2.csv                                 https://ambassador/load-testing/base/
adjust_ambassador USE_NOOP_AUTH=http
run_test oss-https2-httpauth.csv                        https://ambassador/load-testing/base/
adjust_ambassador USE_NOOP_AUTH=grpc
run_test oss-https2-grpcauth.csv                        https://ambassador/load-testing/base/
adjust_ambassador USE_NOOP_AUTH='' USE_PRO_RATELIMIT=true
run_test pro-rlonly-https2-rl-minute.csv                https://ambassador/load-testing/rl-minute/
run_test pro-rlonly-https2-rl-second.csv                https://ambassador/load-testing/rl-second/
adjust_ambassador USE_NOOP_AUTH='' USE_PRO_AUTH=true
run_test pro-https2-base.csv                            https://ambassador/load-testing/base/
run_test pro-https2-rl-minute.csv                       https://ambassador/load-testing/rl-minute/
run_test pro-https2-rl-second.csv                       https://ambassador/load-testing/rl-second/
run_test pro-https2-filter-jwt.csv                      https://ambassador/load-testing/filter-jwt/
