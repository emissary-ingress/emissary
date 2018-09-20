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
	@go test ../...

vendor:
	@echo " >>> installing dependencies"
	@dep ensure -vendor-only

format:
	@echo " >>> running format"
	go fmt ./...

tools:
	@command -v dep >/dev/null ; if [ $$? -ne 0 ]; then \
		echo " >>> installing go dep"; \
		curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh; \
	fi

# For CI use.
docker-login:
	@if [ -z $(DOCKER_USERNAME) ]; then echo 'DOCKER_USERNAME not defined'; exit 1; fi
	@if [ -z $(DOCKER_PASSWORD) ]; then echo 'DOCKER_PASSWORD not defined'; exit 1; fi

	@printf "$(DOCKER_PASSWORD)" | docker login -u="$(DOCKER_USERNAME)" --password-stdin $(DOCKER_REGISTRY)