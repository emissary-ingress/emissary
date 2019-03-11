TAG=10

all: xds echo backend client

.PHONY: backend 

backend: backend.build backend.push

backend.build:
	@echo " ---> building kat-backend image"
	@docker build . -t quay.io/datawire/kat-backend:${TAG}
	
backend.push:
	@echo " ---> pushing kat-backend image"
	@docker push quay.io/datawire/kat-backend:${TAG}


.PHONY: xds 

xds: xds.clean xds.generate 

xds.clean:
	@echo " ---> deleting generated XDS code"
	rm -rf xds/envoy && mkdir xds/envoy

xds.generate:	
	@echo " ---> generating Envoy XDS code"
	@docker build -f ${PWD}/xds/Dockerfile -t envoy-api-build .
	@docker run -it -v ${PWD}/xds/envoy:/envoy envoy-api-build:latest


.PHONY: echo 

echo: echo.clean echo.generate 

echo.clean:
	@echo " ---> deleting generated service code"
	@rm -rf $(PWD)/echo/echo.pb.go

echo.generate:	
	@echo " ---> generating echo service code"
	@docker build -f $(PWD)/echo/Dockerfile -t echo-api-build .
	@docker run -it -v $(PWD)/echo/:/echo echo-api-build:latest


.PHONY: client 

client: client.clean client.build-docker client.build 

client.clean:
	@echo " ---> deleting binaries"
	@rm -rf bin && mkdir bin

client.build-docker:
	@docker build -f $(PWD)/client/Dockerfile -t kat-client-build .	

client.build:	
	@echo " ---> building code"
	@docker run -it --rm -v $(PWD)/client/bin/:/usr/local/tmp/ kat-client-build:latest



.PHONY: sandbox

# For calling the services with kat-client: $ client/bin/client_{OS}_amd64 --input urls.json

sandbox.bridge:
	@echo " ---> cleaning gRPC-Bridge sandbox"
	@cd sandbox/grpc_bridge && docker-compose stop && docker-compose rm -f
	@echo " ---> starting gRPC-Bridge sandbox"
	@cd sandbox/grpc_bridge && docker-compose up --force-recreate --abort-on-container-exit --build

sandbox.http-auth:
	@echo " ---> cleaning HTTP auth sandbox"
	@cd sandbox/http_auth && docker-compose stop && docker-compose rm -f
	@echo " ---> starting HTTP auth sandbox"
	@cd sandbox/http_auth && docker-compose up --force-recreate --abort-on-container-exit --build

sandbox.web:
	@echo " ---> cleaning gRPC web sandbox"
	@cp -R echo/*.js sandbox/grpc_web/
	@cd sandbox/grpc_web && npm install && npx webpack
	@cd sandbox/grpc_web && docker-compose stop && docker-compose rm -f
	@echo " ---> starting gRPC web sandbox"
	@cd sandbox/grpc_web && docker-compose up --force-recreate --abort-on-container-exit --build
