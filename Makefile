DOCKER_REGISTRY ?= localhost:31000
DOCKER_IMAGE = $(DOCKER_REGISTRY)/amb-sidecar-plugin:$(shell git describe --tags --always --dirty)

APRO_VERSION = 0.10.0

apro-abi@%.txt:
	curl --fail -o $@ https://s3.amazonaws.com/datawire-static-files/apro-abi/apro-abi@$(APRO_VERSION).txt
%.mk: %.txt
	{ \
		sed -n 's/^# *_*/APRO_/p' < $<; \
		echo APRO_GOENV=$$(sed -En 's/^# *([A-Z])/\1/p' < $<); \
	} > $@
%.pkgs.txt: %.txt
	grep -v '^#' < $< > $@
-include apro-abi@$(APRO_VERSION).mk

go.DOCKER_IMAGE = golang:$(APRO_GOVERSION)$(if $(filter 2,$(words $(subst ., ,$(APRO_GOVERSION)))),.0)

# Since the GOPATH must match amb-sidecar, we *always* compile in
# Docker, so that we can put it at an arbitrary path without fuss.
go.GOBUILD  = docker exec -i $(shell docker ps -q -f label=component=plugin-builder) go build -trimpath

container.ID = $(shell docker ps -q -f label=component=plugin-builder)

all: .docker.stamp
.PHONY: all

.var.APRO_VERSION: .var.%: FORCE
	@echo $($*) > .tmp$@ && if cmp -s $@ .tmp$@; then cp -f .tmp$@ $@; else rm -f .tmp$@ || true; fi
Dockerfile: Dockerfile.in .var.APRO_VERSION
	sed 's,@APRO_VERSION@,$(APRO_VERSION),' < $< > $@
.docker.stamp: $(patsubst %.go,%.so,$(wildcard *.go)) Dockerfile
	docker build -t $(DOCKER_IMAGE) .
	date > $@

push: .docker.stamp
	docker push $(DOCKER_IMAGE)
.PHONY: push

download-go:
	go list ./...
download-docker:
	docker pull $(go.DOCKER_IMAGE)
.PHONY: download-go download-docker

build-container:
ifeq "$(container.ID)" ""
	docker build -t plugin-builder --build-arg CUR_DIR=$(CURDIR) --build-arg AES_GOVERSION=$(APRO_GOVERSION)$(if $(filter 2,$(words $(subst ., ,$(APRO_GOVERSION)))),.0) --build-arg UID=$(shell id -u) build/
	docker run --rm -d --env-file=${CURDIR}/apro-abi@$(APRO_VERSION).mk plugin-builder
endif

sync: build-container
  # rsync -e 'docker exec -i' -r $$(go env GOPATH)/pkg/mod/cache/download $(container.ID):/mnt/goproxy
	rsync --exclude-from=${CURDIR}/build/sync-excludes.txt -e 'docker exec -i' -r . $(container.ID):$(CURDIR)
	rsync -e 'docker exec -i' -r $(shell go env GOPATH)/pkg/mod/cache/download/ $(container.ID):/mnt/goproxy/

.common-pkgs.txt: apro-abi@$(APRO_VERSION).pkgs.txt download-go
	@bash -c 'comm -12 <(go list -m all|cut -d" " -f1|sort) <(< $< cut -d" " -f1|sort)' > $@
version-check: .common-pkgs.txt apro-abi@$(APRO_VERSION).pkgs.txt
	@bash -c 'diff -u <(grep -F -f $< apro-abi@$(APRO_VERSION).pkgs.txt) <(go list -m all | grep -F -f $<)' || { \
		printf '\nKey:\n  -APro version\n  +Plugin version\n\nERROR: dependency versions do not match APro\n\n'; \
		false; \
	}
.PHONY: version-check

%.so: %.go download-go download-docker version-check sync
	$(go.GOBUILD) -buildmode=plugin -o $@ $<
	rsync -e 'docker exec -i' -r $(container.ID):${CURDIR}/ .

clean:
	rm -f -- *.so .docker.stamp .common-pkgs.txt .tmp.* .var.* Dockerfile apro-abi@*
	docker kill $(container.ID)
.PHONY: clean

.DELETE_ON_ERROR:
.PHONY: FORCE
