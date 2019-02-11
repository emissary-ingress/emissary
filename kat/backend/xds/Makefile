.PHONY: xds 

xds: xds.build-docker xds.clean xds.generate 

xds.build-docker:
	@docker build -t envoy-api-build .

xds.clean:
	@rm -rf envoy && mkdir envoy

xds.generate:	
	@docker run -it -v ${PWD}/envoy:/envoy envoy-api-build:latest
