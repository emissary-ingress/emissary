# This GOPATH stuff is a workaround to allow this repo to work no
# matter where you check it out. Not sure how well it will workout,
# but apparently GOPATH is going away in the latest go release, so
# hopefully it will be short lived.

GOPATH=$(PWD)/build
GO=GOPATH=$(GOPATH) go

all: ambex

vendor:
	glide install

build:
	mkdir build
	ln -s ../vendor build/src
	ln -s ../main.go vendor/main.go

ambex: main.go build vendor
	$(GO) build -o ambex build/src/main.go

clean:
	rm -rf ambex build vendor/main.go

clobber: clean
	rm -rf vendor

ENVOY_IMAGE=envoyproxy/envoy:28d5f4118d60f828b1453cd8ad25033f2c8e38ab

image:
	docker build --build-arg ENVOY_IMAGE=$(ENVOY_IMAGE) . -t bootstrap_image

run: image
	docker run --init --net=host --rm --name ambex-envoy -it bootstrap_image
