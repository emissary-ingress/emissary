# This GOPATH stuff is a workaround to allow this repo to work no
# matter where you check it out. Not sure how well it will workout,
# but apparently GOPATH is going away in the latest go release, so
# hopefully it will be short lived.

GOPATH=$(PWD)/build
GO=GOPATH=$(GOPATH) go

VERSION=$(shell git describe --tags --always)

all: ambex

format:
	gofmt -w -s main.go
.PHONY: format

vendor:
	glide install

build: vendor
	mkdir build
	ln -s ../vendor build/src
	ln -s ../main.go vendor/main.go

# ldflags "-s -w" strips binary
# ldflags -X injects version into binary
# See `go tool link --help` for more info
ambex: main.go build vendor
	$(GO) build \
		--ldflags "-X main.Version=${VERSION}" \
		-o ambex build/src/main.go

clean:
	rm -rf ambex ambex_for_image build vendor/main.go
	docker rmi -f bootstrap_image

clobber: clean
	rm -rf vendor

ENVOY_IMAGE=envoyproxy/envoy:28d5f4118d60f828b1453cd8ad25033f2c8e38ab

ambex_for_image: main.go build vendor
	CGO_ENABLED=0 GOOS=linux $(GO) build \
		--ldflags "-s -w \
		-X main.Version=${VERSION}" \
		-o ambex_for_image build/src/main.go

image: ambex_for_image bootstrap-ads.yaml example
	docker build --build-arg ENVOY_IMAGE=$(ENVOY_IMAGE) . -t bootstrap_image

run: image
	docker run --init --net=host --rm --name ambex-envoy -it bootstrap_image

# For fully in-Docker demo

run_envoy: image
	docker run --init -p8080:8080 --rm --name ambex-envoy -it bootstrap_image

run_ambex:
	docker exec -it -w /application ambex-envoy ./ambex -watch example

