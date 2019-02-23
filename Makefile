TAG=9

.PHONY: backend 

all: xds echo backend

backend: backend.build backend.push

backend.build:
	@echo " ---> building kat-backend image"
	@GO111MODULE=on CGO_ENABLED=0 go build -o bin/kat-server
	@docker build . -t quay.io/datawire/kat-backend:${TAG}
	@rm -rf bin
	
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