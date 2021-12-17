#
# Intro

help:
	@echo 'Usage:'
	@echo '  make help'
	@echo '  make check'
	@echo '  make envconfig.cov.html'
	@echo '  make lint'
	@echo '  make go-mod-tidy'
.PHONY: help

.SECONDARY:

#
# Test

envconfig.cov: check
	test -e $@
	touch $@
check:
	go test -count=1 -coverprofile=envconfig.cov -race ./...
.PHONY: check

%.cov.html: %.cov
	go tool cover -html=$< -o=$@

#
# Lint

lint: tools/bin/golangci-lint
	tools/bin/golangci-lint run ./...
.PHONY: lint

#
# Tools

tools/bin/%: tools/src/%/pin.go tools/src/%/go.mod
	cd $(<D) && GOOS= GOARCH= go build -o $(abspath $@) $$(sed -En 's,^import "(.*)".*,\1,p' pin.go)

#
# `go mod tidy`

go-mod-tidy: .go-mod-tidy/. $(patsubst %/go.mod,.go-mod-tidy/%,$(wildcard tools/src/*/go.mod))
.PHONY: go-mod-tidy

.go-mod-tidy/%: %/go.mod
	rm -f $*/go.sum
	cd $* && GOFLAGS=-mod=mod go mod tidy
	cd $* && GOFLAGS=-mod=mod go mod vendor
	rm -rf $*/vendor
.PHONY: .go-mod-tidy/%
