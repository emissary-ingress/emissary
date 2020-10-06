# Must be included *after* generate.mk, because we use GOHOSTOS and GOHOSTARCH defined there.

# The version number of golangci-lint is controllers in a go.mod file
tools/golangci-lint = $(OSS_HOME)/bin_$(GOHOSTOS)_$(GOHOSTARCH)/golangci-lint
$(tools/golangci-lint): $(OSS_HOME)/build-aux/bin-go/golangci-lint/go.mod
	mkdir -p $(@D)
	cd $(<D) && go build -o $@ github.com/golangci/golangci-lint/cmd/golangci-lint

lint/go-dirs = $(OSS_HOME)

lint: $(tools/golangci-lint)
	@PS4=; set -x; r=0; { \
		for dir in $(lint/go-dirs); do \
			(cd $$dir && $(tools/golangci-lint) run ./...) || r=$?; \
		done; \
		exit $$r; \
	}
.PHONY: lint

format: $(tools/golangci-lint)
	@PS4=; set -x; { \
		for dir in $(lint/go-dirs); do \
			(cd $$dir && $(tools/golangci-lint) run ./...) || true; \
		exit $$r; \
	}
.PHONY: format
