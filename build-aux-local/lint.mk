# Must be included *after* generate.mk, because we use GOHOSTOS and GOHOSTARCH defined there.

# The version number of golangci-lint is controllers in a go.mod file
tools/golangci-lint = $(OSS_HOME)/bin_$(GOHOSTOS)_$(GOHOSTARCH)/golangci-lint
$(tools/golangci-lint): $(OSS_HOME)/build-aux/bin-go/golangci-lint/go.mod
	mkdir -p $(@D)
	cd $(<D) && go build -o $@ github.com/golangci/golangci-lint/cmd/golangci-lint

lint/go-dirs = $(OSS_HOME)

lint:
	@PS4=; set +x; r=0; { \
		printf "$(CYN)==>$(END) Linting $(BLU)Go$(END)...\n" ;\
		go_status=FAIL; go_color="$(RED)" ;\
		if $(MAKE) golint; then go_status=OK; go_color="$(GRN)"; fi ;\
		\
		printf "$(CYN)==>$(END) Linting $(BLU)Python$(END)...\n" ;\
		python_status=FAIL; python_color="$(RED)" ;\
		if $(MAKE) mypy; then python_status=OK; python_color="$(GRN)"; fi ;\
		\
		printf "$(CYN)==>$(END) $(BLU)Go$(END) lint $${go_color}$${go_status}$(END)\n" ;\
		printf "$(CYN)==>$(END) $(BLU)Python$(END) lint $${python_color}$${python_status}$(END)\n" ;\
		test \( "$$go_status" = "OK" \) -a \( "$$python_status" = "OK" \) ;\
		exit $$? ;\
	}
.PHONY: lint

golint: $(tools/golangci-lint)
	@PS4=; set -x; r=0; { \
		for dir in $(lint/go-dirs); do \
			(cd $$dir && $(tools/golangci-lint) run ./...) || r=$$?; \
		done; \
		exit $$r; \
	}
.PHONY: golint

format: $(tools/golangci-lint)
	@PS4=; set -x; { \
		for dir in $(lint/go-dirs); do \
			(cd $$dir && $(tools/golangci-lint) run --fix ./...) || true; \
		done; \
	}
.PHONY: format
