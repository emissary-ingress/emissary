echo "Building images"

set -x
docker build -q -t dwflynn/demo:1.0.0 --build-arg VERSION=1.0.0 demo-service
docker build -q -t dwflynn/demo:2.0.0 --build-arg VERSION=2.0.0 demo-service
docker build -q -t dwflynn/demo:1.0.0tls --build-arg VERSION=1.0.0 --build-arg TLS=--tls demo-service
docker build -q -t dwflynn/demo:2.0.0tls --build-arg VERSION=2.0.0 --build-arg TLS=--tls demo-service
docker build -q -t dwflynn/auth:0.0.1 auth-service
docker build -q -t dwflynn/auth:0.0.1tls --build-arg TLS=--tls auth-service
docker build -q -t dwflynn/stats-test:0.0.1 stats-test-service
docker build -q -t dwflynn/grpc-service:0.0.1 grpc-service

# seriously? there's no docker push --quiet???
docker push dwflynn/demo:1.0.0 | python linify.py push.log
docker push dwflynn/demo:2.0.0 | python linify.py push.log
docker push dwflynn/demo:1.0.0tls | python linify.py push.log
docker push dwflynn/demo:2.0.0tls | python linify.py push.log
docker push dwflynn/auth:0.0.1 | python linify.py push.log
docker push dwflynn/auth:0.0.1tls | python linify.py push.log
docker push dwflynn/stats-test:0.0.1 | python linify.py push.log
docker push dwflynn/grpc-service:0.0.1 | python linify.py push.log
set +x
