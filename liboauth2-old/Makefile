export GO111MODULE = on

all: lint test.cov.html test.cov.func.txt

build:
	go build ./...
test: test.cov
test.cov: build
	go test -coverprofile=test.cov ./...
lint: test
	golangci-lint run ./...
.PHONY: build test lint

%.cov.html: %.cov
	go tool cover -html $< -o $@
%.cov.func.txt: %.cov
	go tool cover -func $< -o $@


go.module = github.com/datawire/liboauth2
go-doc: .gopath
	{ \
		while sleep 1; do \
			$(MAKE) --quiet .gopath/src/$(go.module); \
		done & \
		trap "kill $$!" EXIT; \
		GOPATH=$(CURDIR)/.gopath godoc -http :8080; \
	}
.PHONY: go-doc

vendor: FORCE
	go mod vendor

.gopath: FORCE vendor
	mkdir -p .gopath/src
	rsync --archive --delete vendor/ .gopath/src/
	$(MAKE) .gopath/src/$(go.module)
.gopath/src/$(go.module): FORCE
	mkdir -p $@
	go list ./... | sed -e 's,^$(go.module),,' -e 's,$$,/*.go,' | rsync --archive --prune-empty-dirs --delete-excluded --include='*/' --include-from=/dev/stdin --exclude='*' ./ $@/

.PHONY: FORCE

clean:
	rm -rf vendor .gopath
	rm -f test.cov test.cov.*
.PHONY: clean
