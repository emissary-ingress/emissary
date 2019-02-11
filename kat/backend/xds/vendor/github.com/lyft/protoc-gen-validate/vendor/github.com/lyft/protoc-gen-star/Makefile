# the name of this package
PKG := $(shell go list .)

.PHONY: install
install: # downloads dependencies (including test deps) for the package
	which glide || (curl https://glide.sh/get | sh)
	glide install

.PHONY: lint
lint: # lints the package for common code smells
	which golint || go get -u github.com/golang/lint/golint
	test -z "$(gofmt -d -s ./*.go)" || (gofmt -d -s ./*.go && exit 1)
	golint -set_exit_status
	go tool vet -all -shadow -shadowstrict *.go

.PHONY: quick
quick: # runs all tests without the race detector or coverage percentage
	go test

.PHONY: tests
tests: # runs all tests against the package with race detection and coverage percentage
	go test -race -cover

.PHONY: cover
cover: # runs all tests against the package, generating a coverage report and opening it in the browser
	go test -race -covermode=atomic -coverprofile=cover.out
	go tool cover -html cover.out -o cover.html
	open cover.html

.PHONY: docs
docs: # starts a doc server and opens a browser window to this package
	(sleep 2 && open http://localhost:6060/pkg/$(PKG)/) &
	godoc -http=localhost:6060

.PHONY: demo
demo: # compiles, installs, and runs the demo protoc-plugin
	go install $(PKG)/testdata/protoc-gen-example
	rm -r ./testdata/generated || true
	mkdir -p ./testdata/generated
	set -e; cd ./testdata/protos; for subdir in `find . -type d -mindepth 1 -maxdepth 1`; do \
		protoc -I . --example_out="plugins:../generated" `find $$subdir -name "*.proto"`; \
	done
