#!/bin/bash
set -euo pipefail

if [[ "$(go env GOOS)" = darwin ]]; then
	docker_localhost=host.docker.internal
else
	docker_localhost=localhost
fi

docker run --init -p8080:8080 --rm -d --name ambex-envoy ${docker_localhost}:31000/bootstrap_image
docker exec -d -w /application ambex-envoy ./ambex -watch example

for i in {1..10}; do
    curl -v localhost:8080/hello && break
    echo "(trial ${i} failed. Sleeping before retrying...)"
    sleep 1
done

docker stop ambex-envoy
