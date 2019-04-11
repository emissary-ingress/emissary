#!/bin/sh
cd /tmp
set -x
max-load --csv-file=00-backend-http1.csv         --step-rps=50 --enable-http2=false  http://load-http-echo/load-testing/base/
max-load --csv-file=01-oss-http1.csv             --step-rps=50 --enable-http2=false  http://ambassador-oss-plaintext/load-testing/base/
max-load --csv-file=02-oss-https1.csv            --step-rps=50 --enable-http2=false https://ambassador-oss-tls/load-testing/base/
max-load --csv-file=03-oss-https2.csv            --step-rps=50                      https://ambassador-oss-tls/load-testing/base/
max-load --csv-file=04-oss-https2-httpauth.csv   --step-rps=50                      https://ambassador-oss-tls-httpauth/load-testing/base/
max-load --csv-file=05-oss-https2-grpcauth.csv   --step-rps=50                      https://ambassador-oss-tls-grpcauth/load-testing/base/
max-load --csv-file=06-pro-https2-base.csv       --step-rps=50                      https://ambassador-pro/load-testing/base/
max-load --csv-file=07-pro-https2-rl-minute.csv  --step-rps=50                      https://ambassador-pro/load-testing/rl-minute/
max-load --csv-file=08-pro-https2-rl-second.csv  --step-rps=50                      https://ambassador-pro/load-testing/rl-second/
max-load --csv-file=09-pro-https2-filter-jwt.csv --step-rps=50                      https://ambassador-pro/load-testing/filter-jwt/
python3 -m http.server
