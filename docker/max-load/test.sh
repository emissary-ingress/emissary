#!/bin/sh
cd /tmp
i=0
set -x

max-load --csv-file=$((i++))-backend-http1.csv         --enable-http2=false  http://load-http-echo/load-testing/base/
max-load --csv-file=$((i++))-oss-http1.csv             --enable-http2=false  http://ambassador-oss-plaintext/load-testing/base/
max-load --csv-file=$((i++))-oss-https1.csv            --enable-http2=false https://ambassador-oss-tls/load-testing/base/
max-load --csv-file=$((i++))-oss-https2.csv                                 https://ambassador-oss-tls/load-testing/base/
max-load --csv-file=$((i++))-oss-https2-httpauth.csv                        https://ambassador-oss-tls-httpauth/load-testing/base/
max-load --csv-file=$((i++))-oss-https2-grpcauth.csv                        https://ambassador-oss-tls-grpcauth/load-testing/base/

max-load --csv-file=$((i++))-pro-https2-base.csv                            https://ambassador-pro/load-testing/base/
max-load --csv-file=$((i++))-pro-https2-rl-minute.csv                       https://ambassador-pro/load-testing/rl-minute/
max-load --csv-file=$((i++))-pro-https2-rl-second.csv                       https://ambassador-pro/load-testing/rl-second/
max-load --csv-file=$((i++))-pro-https2-filter-jwt.csv                      https://ambassador-pro/load-testing/filter-jwt/

python3 -m http.server
