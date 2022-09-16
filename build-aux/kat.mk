# ------------------------------------------------------------------------------
# KAT docker-compose sandbox
# ------------------------------------------------------------------------------

tools/sandbox/http_auth/docker-compose.yml tools/sandbox/grpc_auth/docker-compose.yml tools/sandbox/grpc_web/docker-compose.yaml: %: %.in kat-server.docker.push.dev
	sed "s,@KAT_SERVER_DOCKER_IMAGE@,$$(cat kat-server.docker.push.dev),g" < $< > $@

tools/sandbox.http-auth: ## In docker-compose: run Ambassador, an HTTP AuthService, an HTTP backend service, and a TracingService
tools/sandbox.http-auth: tools/sandbox/http_auth/docker-compose.yml
	@echo " ---> cleaning HTTP auth tools/sandbox"
	@cd tools/sandbox/http_auth && docker-compose stop && docker-compose rm -f
	@echo " ---> starting HTTP auth tools/sandbox"
	@cd tools/sandbox/http_auth && docker-compose up --force-recreate --abort-on-container-exit --build
.PHONY: tools/sandbox.http-auth

tools/sandbox.grpc-auth: ## In docker-compose: run Ambassador, a gRPC AuthService, an HTTP backend service, and a TracingService
tools/sandbox.grpc-auth: tools/sandbox/grpc_auth/docker-compose.yml
	@echo " ---> cleaning gRPC auth tools/sandbox"
	@cd tools/sandbox/grpc_auth && docker-compose stop && docker-compose rm -f
	@echo " ---> starting gRPC auth tools/sandbox"
	@cd tools/sandbox/grpc_auth && docker-compose up --force-recreate --abort-on-container-exit --build
.PHONY: tools/sandbox.grpc-auth

tools/sandbox.web: ## In docker-compose: run Ambassador with gRPC-web enabled, and a gRPC backend service
tools/sandbox.web: tools/sandbox/grpc_web/docker-compose.yaml
tools/sandbox.web: tools/sandbox/grpc_web/echo_grpc_web_pb.js tools/sandbox/grpc_web/echo_pb.js
	@echo " ---> cleaning gRPC web tools/sandbox"
	@cd tools/sandbox/grpc_web && npm install && npx webpack
	@cd tools/sandbox/grpc_web && docker-compose stop && docker-compose rm -f
	@echo " ---> starting gRPC web tools/sandbox"
	@cd tools/sandbox/grpc_web && docker-compose up --force-recreate --abort-on-container-exit --build
.PHONY: tools/sandbox.web
