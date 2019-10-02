go.PLATFORMS    = linux_amd64 darwin_amd64
export CGO_ENABLED = 0

include ../../build-aux/go-mod.mk
include ../../build-aux/go-version.mk
include ../../build-aux/docker.mk
include ../../build-aux/help.mk

.DEFAULT_GOAL = help

clean: docker.docker.clean
# Files made by older versions.  Remove the tail of this list when the
# commit making the change gets far enough in to the past.
#
# 2019-01-23
	rm -f ambex ambex_for_image

image: ## docker build
image: bootstrap_image.docker
.PHONY: image

bootstrap_image.docker: bootstrap_image/ambex
bootstrap_image/%: bin_linux_amd64/%
	cp $< $@

run: ## ???
run: bootstrap_image.docker
	docker run --init --net=host --rm --name ambex-envoy -it $$(head -n1 $<)
.PHONY: run

# For fully in-Docker demo

run_envoy: ## ???
run_envoy: bootstrap_image.docker
	docker run --init -p8080:8080 --rm --name ambex-envoy -it $$(head -n1 $<)

run_ambex: ## ???
	docker exec -it -w /application ambex-envoy ./ambex -watch example

.PHONY: run_envoy run_ambex

# Configuration of sorts
