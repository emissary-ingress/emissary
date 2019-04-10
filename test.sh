#!/bin/sh
make apply
export KUBECONFIG=$PWD/build-aux/ambassador-pro.knaut
./bin_linux_amd64/max-load --csv-file=01-baseline.csv  --step-rps=50 nodeport+https://ambassador-baseline.no-pro/load-testing/base/
./bin_linux_amd64/max-load --csv-file=02-grpc-auth.csv --step-rps=50 nodeport+https://ambassador-grpc.no-pro/load-testing/base/
./bin_linux_amd64/max-load --csv-file=03-http-auth.csv --step-rps=50 nodeport+https://ambassador-http.no-pro/load-testing/base/
./bin_linux_amd64/max-load --csv-file=03-pro-auth.csv  --step-rps=50 nodeport+https://ambassador.default/load-testing/base/
