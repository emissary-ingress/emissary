DOCKER_REGISTRY ?= localhost:31000
DOCKER_IMAGE = $(DOCKER_REGISTRY)/amb-sidecar-plugin:$(shell git describe --tags --always --dirty)

# The Go version must exactly match what was used to compile the amb-sidecar
GO_VERSION = 1.11.4

ifeq ($(shell go version),go version go$(GO_VERSION) linux/amd64)
RUN =
else
RUN = docker run --rm --volume $(CURDIR):$(CURDIR):rw --workdir $(CURDIR) golang:$(GO_VERSION)
endif

all: .docker.stamp
.docker.stamp: $(patsubst %.go,%.so,$(wildcard *.go)) Dockerfile
	docker build -t $(DOCKER_IMAGE) .
	date > $@

push: .docker.stamp
	docker push $(DOCKER_REGISTRY)

%.so: %.go
	$(RUN) env GOOS=linux GOARCH=amd64 CGO_ENABLED=1 GO111MODULE=on go build -buildmode=plugin -o $@ $<

clean:
	rm -f -- *.so .docker.stamp

.PHONY: all push clean
.DELETE_ON_ERROR:
