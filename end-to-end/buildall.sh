echo "Building images"

set -x
docker build -q -t dwflynn/demo:1.0.0 --build-arg VERSION=1.0.0 demo-service
docker build -q -t dwflynn/demo:2.0.0 --build-arg VERSION=2.0.0 demo-service
docker build -q -t dwflynn/demo:1.0.0tls --build-arg VERSION=1.0.0 --build-arg TLS=--tls demo-service
docker build -q -t dwflynn/demo:2.0.0tls --build-arg VERSION=2.0.0 --build-arg TLS=--tls demo-service
docker build -q -t dwflynn/auth:0.0.1 auth-service
docker build -q -t dwflynn/auth:0.0.1tls --build-arg TLS=--tls auth-service

# seriously? there's no docker push --quiet???
docker push dwflynn/demo:1.0.0
docker push dwflynn/demo:2.0.0
docker push dwflynn/demo:1.0.0tls
docker push dwflynn/demo:2.0.0tls
docker push dwflynn/auth:0.0.1
docker push dwflynn/auth:0.0.1tls
set +x
