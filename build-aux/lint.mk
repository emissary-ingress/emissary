include build-aux/tools.mk

lint/go-dirs = $(OSS_HOME)

lint:
	@PS4=; set +ex; r=0; { \
		printf "$(CYN)==>$(END) Linting $(BLU)Go$(END)...\n"; \
		go_status=0; $(MAKE) golint || { go_status=$$?; r=$$go_status; }; \
		\
		printf "$(CYN)==>$(END) Linting $(BLU)Python$(END)...\n"; \
		py_status=0; $(MAKE) mypy || { py_status=$$?; r=$$py_status; }; \
		\
		printf "$(CYN)==>$(END) Linting $(BLU)Helm$(END)...\n"; \
		helm_status=0; $(MAKE) lint-chart || { helm_status=$$?; r=$$helm_status; }; \
		\
		set +x; \
		printf "$(CYN)==>$(END) $(BLU)Go$(END)      lint $$(if [[ $$go_status     == 0 ]]; then printf "$(GRN)OK"; else printf "$(RED)FAIL"; fi)$(END)\n"; \
		printf "$(CYN)==>$(END) $(BLU)Python$(END)  lint $$(if [[ $$py_status     == 0 ]]; then printf "$(GRN)OK"; else printf "$(RED)FAIL"; fi)$(END)\n"; \
		printf "$(CYN)==>$(END) $(BLU)Helm$(END)    lint $$(if [[ $$helm_status   == 0 ]]; then printf "$(GRN)OK"; else printf "$(RED)FAIL"; fi)$(END)\n"; \
		set -x; \
		\
		exit $$r; \
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

lint-chart: $(tools/ct) $(chart_dir)
	cd $(chart_dir) && $(abspath $(tools/ct)) lint --config=./ct.yaml
.PHONY: lint-chart
