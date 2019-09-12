TAG=v1.6.0

# Target          ## ## Description
################# ## ## #########################
help              :  ## Display this help text
	@grep '[#][#]' $(MAKEFILE_LIST)
.PHONY: help

################# ## ## #########################
build             :  ## Build everything
push              :  ## Push docker images
clean             :: ## Remove everything
.PHONY: build push clean

%.docker.clean::
	if [ -e $*.docker ]; then docker image rm $$(cat $*.docker) || true; fi
.PHONY: %.docker.clean

.PHONY: FORCE

#
# Echo service definition

# What are we going to do?
build:
push:
clean:: echo/echo.docker.clean
	rm -f echo/echo.pb.go echo/echo_grpc_web_pb.js echo/echo_pb.js

# How are we going to do it?
#
# TODO: We install `protoc` and the required protoc plugins in a
# docker image, instead of installing them in ./bin/ or something.
# Fix that.
echo/echo.docker: echo/Dockerfile FORCE
	docker build --iidfile=$@ --file=$< .
%.pb.go %_grpc_web_pb.js %_pb.js: %.docker %.proto
	docker run --rm --volume=$(CURDIR)/echo:/echo $$(cat $<)

#
# Backend Docker image

# What are we going to do?
build: backend.docker
push: backend.docker.push
clean:: backend.docker.clean

# How are we going to do it?
backend.docker: Dockerfile echo/echo.pb.go echo/echo_grpc_web_pb.js echo/echo_pb.js FORCE
	docker build --iidfile=$@ --tag=quay.io/datawire/kat-backend:${TAG} .
backend.docker.push: %.docker.push: %.docker
	docker push quay.io/datawire/kat-backend:${TAG}
.PHONY: backend.docker.push


# Client binary executables

# What are we going to do?
build: client/bin/client_darwin_amd64 client/bin/client_linux_amd64
push:
clean:: client/client.docker.clean
	rm -rf client/bin

# How are we going to do it?
client/bin:
	mkdir $@
client/bin/client_%_amd64: echo/echo.pb.go FORCE | client/bin
	GO111MODULE=on CGO_ENABLED=0 GOOS=$* GOARCH=amd64 go build -o $(abspath $@) ./client


# docker-compose sandbox

build: sandbox/grpc_web/echo_grpc_web_pb.js sandbox/grpc_web/echo_pb.js
buld: sandbox/http_auth/docker-compose.yml sandbox/grpc_auth/docker-compose.yml sandbox/grpc_web/docker-compose.yaml
push:
clean::
	rm -f sandbox/grpc_web/echo_grpc_web_pb.js sandbox/grpc_web/echo_pb.js
	rm -f sandbox/http_auth/docker-compose.yml sandbox/grpc_auth/docker-compose.yml sandbox/grpc_web/docker-compose.yaml

sandbox/http_auth/docker-compose.yml sandbox/grpc_auth/docker-compose.yml sandbox/grpc_web/docker-compose.yaml: %: %.in backend.docker
	sed 's/@TAG@/$(TAG)/g' < $< > $@
sandbox/grpc_web/echo%: echo/echo%
	cp $< $@

# For calling the services with kat-client: $ client/bin/client_{OS}_amd64 --input urls.json

################# ## ## #########################
sandbox.http-auth :  ## In docker-compose: run Ambassador, an HTTP AuthService, an HTTP backend service, and a TracingService
sandbox.http-auth: sandbox/http_auth/docker-compose.yml
	@echo " ---> cleaning HTTP auth sandbox"
	@cd sandbox/http_auth && docker-compose stop && docker-compose rm -f
	@echo " ---> starting HTTP auth sandbox"
	@cd sandbox/http_auth && docker-compose up --force-recreate --abort-on-container-exit --build
.PHONY: sandbox.http-auth

sandbox.grpc-auth :  ## In docker-compose: run Ambassador, a gRPC AuthService, an HTTP backend service, and a TracingService
sandbox.grpc-auth: sandbox/grpc_auth/docker-compose.yml
	@echo " ---> cleaning gRPC auth sandbox"
	@cd sandbox/grpc_auth && docker-compose stop && docker-compose rm -f
	@echo " ---> starting gRPC auth sandbox"
	@cd sandbox/grpc_auth && docker-compose up --force-recreate --abort-on-container-exit --build
.PHONY: sandbox.grpc-auth

sandbox.web       :  ## In docker-compose: run Ambassador with gRPC-web enabled, and a gRPC backend service
sandbox.web: sandbox/grpc_web/docker-compose.yaml
sandbox.web: sandbox/grpc_web/echo_grpc_web_pb.js sandbox/grpc_web/echo_pb.js
	@echo " ---> cleaning gRPC web sandbox"
	@cd sandbox/grpc_web && npm install && npx webpack
	@cd sandbox/grpc_web && docker-compose stop && docker-compose rm -f
	@echo " ---> starting gRPC web sandbox"
	@cd sandbox/grpc_web && docker-compose up --force-recreate --abort-on-container-exit --build
.PHONY: sandbox.web
