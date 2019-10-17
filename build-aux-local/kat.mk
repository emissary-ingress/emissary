# ------------------------------------------------------------------------------
# gRPC bindings for KAT
# ------------------------------------------------------------------------------

GRPC_WEB_VERSION = 1.0.3
GRPC_WEB_PLATFORM = $(GOOS)-x86_64

bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc-gen-grpc-web: $(var.)GRPC_WEB_VERSION $(var.)GRPC_WEB_PLATFORM
	mkdir -p $(@D)
	curl -o $@ -L --fail https://github.com/grpc/grpc-web/releases/download/$(GRPC_WEB_VERSION)/protoc-gen-grpc-web-$(GRPC_WEB_VERSION)-$(GRPC_WEB_PLATFORM)
	chmod 755 $@

pkg/api/kat/echo.pb.go: api/kat/echo.proto bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc-gen-gogofast
	mkdir -p $(@D)
	./bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc \
		--proto_path=$(CURDIR)/api/kat \
		--plugin=$(CURDIR)/bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc-gen-gogofast --gogofast_out=plugins=grpc:$(@D) \
		$(CURDIR)/$<

tools/sandbox/grpc_web/echo_grpc_web_pb.js: api/kat/echo.proto bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc-gen-grpc-web
	./bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc \
		--proto_path=$(CURDIR)/api/kat \
		--plugin=$(CURDIR)/bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc-gen-grpc-web --grpc-web_out=import_style=commonjs,mode=grpcwebtext:$(@D) \
		$(CURDIR)/$<

tools/sandbox/grpc_web/echo_pb.js: api/kat/echo.proto bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc
	./bin_$(GOHOSTOS)_$(GOHOSTARCH)/protoc \
		--proto_path=$(CURDIR)/api/kat \
		--js_out=import_style=commonjs:$(@D) \
		$(CURDIR)/$<

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
