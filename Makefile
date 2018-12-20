ifeq ("$(GOPATH)","")
$(error GOPATH must be set)
endif

SHELL := /bin/bash
GOREPO := ${GOPATH}/src/github.com/lyft/ratelimit

.PHONY: bootstrap
bootstrap:
	script/install-glide
	script/install-protoc
	glide install

.PHONY: bootstrap_tests
bootstrap_tests:
	cd ./vendor/github.com/golang/mock/mockgen && go install

.PHONY: docs_format
docs_format:
	script/docs_check_format

.PHONY: fix_format
fix_format:
	script/docs_fix_format
	go fmt $(shell glide nv)

.PHONY: check_format
check_format: docs_format
	@gofmt -l $(shell glide nv | sed 's/\.\.\.//g') | tee /dev/stderr | read && echo "Files failed gofmt" && exit 1 || true

.PHONY: compile
compile: proto
	mkdir -p ${GOREPO}/bin
	cd ${GOREPO}/src/service_cmd && go build -o ratelimit ./ && mv ./ratelimit ${GOREPO}/bin
	cd ${GOREPO}/src/client_cmd && go build -o ratelimit_client ./ && mv ./ratelimit_client ${GOREPO}/bin
	cd ${GOREPO}/src/config_check_cmd && go build -o ratelimit_config_check ./ && mv ./ratelimit_config_check ${GOREPO}/bin

.PHONY: tests_unit
tests_unit: compile
	go test $(shell glide nv)

.PHONY: tests
tests: compile
	go test $(shell glide nv) -tags=integration

.PHONY: proto
proto:
	script/generate_proto

.PHONY: docker
docker: tests
	docker build . -t lyft/ratelimit:`git rev-parse HEAD`
