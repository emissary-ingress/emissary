export GO111MODULE = on

lint:
	go build ./...
	golangci-lint run ./...
	golint -set_exit_status ./...
	unused -exported ./...
.PHONY: lint

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
