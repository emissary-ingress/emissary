# This GOPATH stuff is a workaround to allow this repo to work no
# matter where you check it out. Not sure how well it will workout,
# but apparently GOPATH is going away in the latest go release, so
# hopefully it will be short lived.

pkg = github.com/datawire/ambex

export GOPATH=$(CURDIR)/.go-workspace
export GOBIN=$(CURDIR)

VERSION=$(shell git describe --tags --always)

all: ambex

format:
	gofmt -w -s main.go
.PHONY: format

vendor: glide.yaml glide.lock
	glide install

# ldflags "-s -w" strips binary
# ldflags -X injects version into binary
# See `go tool link --help` for more info
ambex: vendor FORCE
	go install \
		--ldflags "-X main.Version=${VERSION}" \
		$(pkg)

clean:
	rm -rf ambex ambex_for_image vendor
	docker rmi -f bootstrap_image

clobber: clean
	rm -rf vendor

.PHONY: clean clobber

ENVOY_IMAGE=envoyproxy/envoy:28d5f4118d60f828b1453cd8ad25033f2c8e38ab

ambex_for_image: vendor FORCE
	CGO_ENABLED=0 GOOS=linux go build \
		--ldflags "-s -w \
		-X main.Version=${VERSION}" \
		-o ambex_for_image $(pkg)

image: ambex_for_image bootstrap-ads.yaml example
	docker build --build-arg ENVOY_IMAGE=$(ENVOY_IMAGE) . -t bootstrap_image

run: image
	docker run --init --net=host --rm --name ambex-envoy -it bootstrap_image

.PHONY: image run

# For fully in-Docker demo

run_envoy: image
	docker run --init -p8080:8080 --rm --name ambex-envoy -it bootstrap_image

run_ambex:
	docker exec -it -w /application ambex-envoy ./ambex -watch example

.PHONY: run_envoy run_ambex

# Configuration of sorts

.SECONDARY:
# The only reason .DELETE_ON_ERROR is off by default is for historical
# compatibility.
.DELETE_ON_ERROR:
# .NOTPARALLEL is important, as having multiple `go install`s going at
# once can corrupt `$(GOPATH)/pkg`.  Setting .NOTPARALLEL is simpler
# than mucking with multi-target pattern rules.
.NOTPARALLEL:
# The $(bins) aren't .PHONY--they're real files that will exist, but
# we should try to update them every run, and let `go` decide if
# they're up-to-date or not, rather than trying to teach Make to do
# it.  So instead, have them depend on a .PHONY target so that they'll
# always be considered out-of-date.
.PHONY: FORCE
