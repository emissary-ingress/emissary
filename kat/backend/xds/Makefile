.PHONY: xds 

xds: xds.build-docker xds.clean xds.generate 

xds.build-docker:
	@docker build -t envoy-api-build .

xds.clean:
	@rm -rf vendor envoy && mkdir vendor envoy

xds.generate:	
	@docker run -it -v ${PWD}/envoy:/envoy -v ${PWD}/vendor:/vendor envoy-api-build:latest
