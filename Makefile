NAME=ambassador-pro

REGISTRY=quay.io
REGISTRY_NAMESPACE=datawire
VERSION=0.0.2
K8S_DIR=scripts

include k8s.make

.SHELL: /bin/bash

.PHONY: run
run: install
	@echo " >>> running oauth server"
	ambassador-oauth 

.PHONY: install
install: tools vendor
	@echo " >>> building"
	@go install ./cmd/...

.PHONY: clean
clean: clean-k8s
	@echo " >>> cleaning compiled objects and binaries"
	@go clean -i ./...

.PHONY: test
test:
	@echo " >>> testing code.."
	@go test ./...

vendor:
	@echo " >>> installing dependencies"
	@dep ensure -vendor-only

format:
	@echo " >>> running format"
	go fmt ./...

check_format:
	@echo " >>> checking format"
	@if [ $$(go fmt $$(go list ./... | grep -v vendor/)) ]; then exit 1; fi

tools:
	@command -v dep >/dev/null ; if [ $$? -ne 0 ]; then \
		echo " >>> installing go dep"; \
		curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh; \
	fi

e2e_build:
	@echo " >>> building docker for e2e testing"
	@/bin/bash -c "cd $(PWD)/e2e && docker build -t e2e/test:latest ." 
	
e2e_test:
	@echo " >>> running e2e tests"
	docker run --rm e2e/test:latest
	