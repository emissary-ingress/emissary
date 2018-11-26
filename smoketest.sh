#!/bin/bash
set -euo pipefail

make image
docker run --init -p8080:8080 --rm -d --name ambex-envoy bootstrap_image
docker exec -d -w /application ambex-envoy ./ambex -watch example

for i in {1..10}; do
    curl -v localhost:8080/hello && break
    echo "(trial ${i} failed. Sleeping before retrying...)"
    sleep 1
done

docker stop ambex-envoy
