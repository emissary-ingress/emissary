DOCKER_REGISTRY ?= localhost:31000
DOCKER_IMAGE = $(DOCKER_REGISTRY)/amb-sidecar-plugin:$(shell git describe --tags --always --dirty)

APRO_VERSION = 0.9.0

apro-abi@%.txt:
	curl --fail -o $@ https://s3.amazonaws.com/datawire-static-files/apro-abi/apro-abi@$(APRO_VERSION).txt
%.mk: %.txt
	{ \
		sed -n 's/^# *_*/APRO_/p' < $<; \
		echo APRO_GOENV = $$(sed -En 's/^# *([A-Z])/\1/p' < $<); \
	} > $@
%.pkgs.txt: %.txt
	grep -v '^#' < $< > $@
-include apro-abi@$(APRO_VERSION).mk

go.DOCKER_IMAGE = golang:$(APRO_GOVERSION)$(if $(filter 2,$(words $(subst ., ,$(APRO_GOVERSION)))),.0)

# Since the GOPATH must match amb-sidecar, we *always* compile in
# Docker, so that we can put it at an arbitrary path without fuss.
go.GOBUILD  = docker run --rm
# Map in the Go module cache so that we don't need to re-download
# things every time.
go.GOBUILD += --volume=$$(go env GOPATH)/pkg/mod/cache/download:/mnt/goproxy:ro --env=GOPROXY=file:///mnt/goproxy
# Simulate running in the current directory as the current user.  That
# UID probably doesn't have access to the container's default
# GOCACHE=/.cache/go-build.
go.GOBUILD += --volume $(CURDIR):$(CURDIR):rw --workdir=$(CURDIR) --user=$$(id -u) --env=GOCACHE=/tmp/go-cache
go.GOBUILD += $(foreach _gopath,$(subst :, ,$(APRO_GOPATH)), --tmpfs=$(_gopath):uid=$$(id -u),gid=$$(id -g),mode=0755,rw )
# Run `go build` mimicking the APro build
go.GOBUILD += $(addprefix --env=,$(APRO_GOENV)) $(go.DOCKER_IMAGE) go build -trimpath

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
	docker push $(DOCKER_REGISTRY)
.PHONY: push

download-go:
	go list ./...
download-docker:
	docker pull $(go.DOCKER_IMAGE)
.PHONY: download-go download-docker

.common-pkgs.txt: apro-abi@$(APRO_VERSION).pkgs.txt download-go
	@bash -c 'comm -12 <(go list -m all|cut -d" " -f1|sort) <(< $< cut -d" " -f1|sort)' > $@
version-check: .common-pkgs.txt apro-abi@$(APRO_VERSION).pkgs.txt
	@bash -c 'diff -u <(grep -F -f $< apro-abi@$(APRO_VERSION).pkgs.txt) <(go list -m all | grep -F -f $<)' || { \
		printf '\nKey:\n  -APro version\n  +Plugin version\n\nERROR: dependency versions do not match APro\n\n'; \
		false; \
	}
.PHONY: version-check

%.so: %.go download-go download-docker version-check
	$(go.GOBUILD) -buildmode=plugin -o $@ $<

clean:
	rm -f -- *.so .docker.stamp .common-pkgs.txt .tmp.* .var.* Dockerfile apro-abi@*
.PHONY: clean

.DELETE_ON_ERROR:
.PHONY: FORCE
