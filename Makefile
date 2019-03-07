DOCKER_REGISTRY ?= localhost:31000
DOCKER_IMAGE = $(DOCKER_REGISTRY)/amb-sidecar-plugin:$(shell git describe --tags --always --dirty)

# These details must match how amb-sidecar was compiled
APRO_GOVERSION = 1.12
APRO_GOPATH = /home/circleci/go
APRO_GOENV = GOPATH=$(APRO_GOPATH) GOOS=linux GOARCH=amd64 CGO_ENABLED=1 GO111MODULE=on
# APRO_PKGFILE stores the output of the following command run from the
# APro source directory:
#
#    GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go list -deps -f='{{if not .Standard}}{{.Module}}{{end}}' ./cmd/amb-sidecar | sort -u | grep -v -e '=>' -e '/apro$'
APRO_PKGFILE = apro-pkgs.txt

# Since the GOPATH must match amb-sidecar, we *always* compile in
# Docker, so that we can put it at an arbitrary path without fuss.
go.GOBUILD  = docker run --rm
# Map in the Go module cache so that we don't need to re-download
# things every time.
go.GOBUILD += --volume=$$(go env GOPATH)/pkg/mod:$(APRO_GOPATH)/pkg/mod:ro
# Simulate running in the current directory as the current user.  That
# UID probably doesn't have access to the container's default
# GOCACHE=/.cache/go-build.
go.GOBUILD += --volume $(CURDIR):$(CURDIR):rw --workdir=$(CURDIR) --user=$$(id -u) --env=GOCACHE=/tmp/go-cache
# Run `go build` mimicking the APro build
go.GOBUILD += $(addprefix --env=,$(APRO_GOENV)) golang:$(APRO_GOVERSION) go build

all: .docker.stamp
.PHONY: all

.docker.stamp: $(patsubst %.go,%.so,$(wildcard *.go)) Dockerfile
	docker build -t $(DOCKER_IMAGE) .
	date > $@

push: .docker.stamp
	docker push $(DOCKER_REGISTRY)
.PHONY: push

download-go:
	go list ./...
.PHONY: download-go

.common-pkgs.txt: $(APRO_PKGFILE) download-go
	@bash -c 'comm -12 <(go list -m all|cut -d" " -f1|sort) <(< $< cut -d" " -f1|sort)' > $@
version-check: .common-pkgs.txt $(APRO_PKGFILE)
	@bash -c 'diff -u <(grep -F -f $< $(APRO_PKGFILE)) <(go list -m all | grep -F -f $<)' || { \
		printf '\nKey:\n  -APro version\n  +Plugin version\n\nERROR: dependency versions do not match APro\n\n'; \
		false; \
	}
.PHONY: version-check

%.so: %.go download-go version-check
	$(go.GOBUILD) -buildmode=plugin -o $@ $<

clean:
	rm -f -- *.so .docker.stamp .common-pkgs.txt
.PHONY: clean

.DELETE_ON_ERROR:
