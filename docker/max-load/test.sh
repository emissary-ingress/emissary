#!/bin/sh
args='--load-max-rps=3000'
args='--load-max-rps=300'

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

cd /tmp
trap 'python3 -m http.server' EXIT
i=0
set -ex
preambassador_cleanup
max-load $args --csv-file=$((i++))-backend-http1.csv         --enable-http2=false  http://load-http-echo/load-testing/base/
adjust_ambassador USE_TLS=false USE_NOOP_AUTH='' USE_PRO_RATELIMIT=false USE_PRO_AUTH=false
max-load $args --csv-file=$((i++))-oss-http1.csv             --enable-http2=false  http://ambassador/load-testing/base/
adjust_ambassador USE_TLS=true
max-load $args --csv-file=$((i++))-oss-https1.csv            --enable-http2=false https://ambassador/load-testing/base/
max-load $args --csv-file=$((i++))-oss-https2.csv                                 https://ambassador/load-testing/base/
adjust_ambassador USE_NOOP_AUTH=http
max-load $args --csv-file=$((i++))-oss-https2-httpauth.csv                        https://ambassador/load-testing/base/
adjust_ambassador USE_NOOP_AUTH=grpc
max-load $args --csv-file=$((i++))-oss-https2-grpcauth.csv                        https://ambassador/load-testing/base/
adjust_ambassador USE_NOOP_AUTH='' USE_PRO_RATELIMIT=true
max-load $args --csv-file=$((i++))-pro-rlonly-https2-rl-minute.csv                https://ambassador/load-testing/rl-minute/
max-load $args --csv-file=$((i++))-pro-rlonly-https2-rl-second.csv                https://ambassador/load-testing/rl-second/
adjust_ambassador USE_NOOP_AUTH='' USE_PRO_AUTH=true
max-load $args --csv-file=$((i++))-pro-https2-base.csv                            https://ambassador/load-testing/base/
max-load $args --csv-file=$((i++))-pro-https2-rl-minute.csv                       https://ambassador/load-testing/rl-minute/
max-load $args --csv-file=$((i++))-pro-https2-rl-second.csv                       https://ambassador/load-testing/rl-second/
max-load $args --csv-file=$((i++))-pro-https2-filter-jwt.csv                      https://ambassador/load-testing/filter-jwt/
