#!/bin/sh
make claim
export KUBECONFIG=$PWD/build-aux/ambassador-pro.knaut
./bin_linux_amd64/max-load --csv-file=00-backend-http1.csv --enable-http2=false nodeport+http://load-http-echo.no-pro/load-testing/base/
#./bin_linux_amd64/max-load --csv-file=01-oss-http1.csv    --enable-http2=false nodeport+http://ambassador.no-pro/load-testing/base/
./bin_linux_amd64/max-load --csv-file=02-oss-https1.csv    --enable-http2=false nodeport+https://ambassador.no-pro/load-testing/base/
./bin_linux_amd64/max-load --csv-file=03-oss-https2.csv                         nodeport+https://ambassador.no-pro/load-testing/base/
./bin_linux_amd64/max-load --csv-file=04-pro-https2-base.csv       nodeport+https://ambassador/load-testing/base/
./bin_linux_amd64/max-load --csv-file=05-pro-https2-rl-minute.csv  nodeport+https://ambassador/load-testing/rl-minute/
./bin_linux_amd64/max-load --csv-file=06-pro-https2-rl-second.csv  nodeport+https://ambassador/load-testing/rl-second/
./bin_linux_amd64/max-load --csv-file=07-pro-https2-filter-jwt.csv nodeport+https://ambassador/load-testing/filter-jwt/
