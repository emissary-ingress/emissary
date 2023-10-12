include build-aux/tools.mk

#
# Go

lint-deps += $(tools/golangci-lint)
lint-goals += lint/go
lint/go: $(tools/golangci-lint)
	$(tools/golangci-lint) run ./...
.PHONY: lint/go

format-goals += format/go
format/go: $(tools/golangci-lint)
	$(tools/golangci-lint) run --fix ./... || true
.PHONY: format/go

#
# Python

lint-deps += $(OSS_HOME)/venv

lint-goals += lint/mypy
lint/mypy: $(OSS_HOME)/venv
	set -e; { \
	  . $(OSS_HOME)/venv/bin/activate; \
	  time mypy \
	    --cache-fine-grained \
	    --ignore-missing-imports \
	    --check-untyped-defs \
	    ./python/; \
	}
.PHONY: lint/mypy
clean: .dmypy.json.rm .mypy_cache.rm-r

lint-goals += lint/black
lint/black: $(OSS_HOME)/venv
	. $(OSS_HOME)/venv/bin/activate && black --check ./python/
.PHONY: lint/black

format-goals += format/black
format/black: $(OSS_HOME)/venv
	. $(OSS_HOME)/venv/bin/activate && black ./python/
.PHONY: format/black

lint-goals += lint/isort
lint/isort: $(OSS_HOME)/venv
	. $(OSS_HOME)/venv/bin/activate && isort --check --diff ./python/
.PHONY: lint/isort

format-goals += format/isort
format/isort: $(OSS_HOME)/venv
	. $(OSS_HOME)/venv/bin/activate && isort ./python/
.PHONY: format/isort

#
# Helm
lint-deps += $(tools/ct) $(chart_dir)
lint-goals += lint/chart
lint/chart: $(tools/ct) $(chart_dir)
	cd $(chart_dir) && $(abspath $(tools/ct)) lint --config=./ct.yaml
.PHONY: lint/chart

#
# All together now

lint-deps: ## (QA) Everything necessary to lint (useful to separate out in the logs)
lint-deps: $(lint-deps)
.PHONY: lint-deps

lint: ## (QA) Run the linters
lint: lint-deps
	@printf "$(GRN)==> $(BLU)Running linters...$(END)\n"
	@{ \
	  r=0; \
	  for goal in $(lint-goals); do \
	    printf " $(BLU)=> $${goal}$(END)\n"; \
	    echo "$(MAKE) $${goal}"; \
	    $(MAKE) "$${goal}" || r=$$?; \
	  done; \
	  exit $$r; \
	}
.PHONY: lint

format: ## (QA) Automatically fix linter complaints
format: $(format-goals)
.PHONY: format
