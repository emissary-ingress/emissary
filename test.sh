#!/bin/sh
make apply
export KUBECONFIG=$PWD/build-aux/ambassador-pro.knaut
./bin_linux_amd64/max-load --csv-file=00-base.csv --step-rps=50 nodeport+https://ambassador.no-pro/load-testing/base/
./bin_linux_amd64/max-load --csv-file=01-grpc.csv --step-rps=50 nodeport+https://ambassador/load-testing/base/
