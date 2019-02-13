DOCKER_IMAGE = localhost:31000/amb-sidecar-plugin:$(shell git describe --tags --always --dirty)

ifeq ($(shell go env GOOS)_$(shell go env GOARCH),linux_amd64)
RUN =
else
RUN = docker run --rm -it --volume $(CURDIR):$(CURDIR):rw --workdir $(CURDIR) golang:1.11.5
endif

all: .docker.stamp
.docker.stamp: example-plugin.so Dockerfile
	docker build -t $(DOCKER_IMAGE) .
	date > $@

example-plugin.so: FORCE
	$(RUN) GOOS=linux GOARCH=amd64 go build -buildmode=plugin -o $@ .

clean:
	rm -f -- *.so .docker.stamp

.PHONY: FORCE all clean
.DELETE_ON_ERROR:
