# Copyright 2018 Datawire. All rights reserved.
#
# Makefile snippet for bits common bits we "always" want.
#
## Eager inputs ##
#  (none)
## Lazy inputs ##
#  (none)
## Outputs ##
#  - Variable: GOOS
#  - Variable: GOARCH
#  - .PHONY Target: all
#  - .PHONY Target: build
#  - .PHONY Target: check
#  - .PHONY Target: lint
#  - .PHONY Target: format
#  - .PHONY Target: clean
#  - .PHONY Target: clobber
## common.mk targets ##
#  (N/A)
ifeq ($(words $(filter $(abspath $(lastword $(MAKEFILE_LIST))),$(abspath $(MAKEFILE_LIST)))),1)
_common.mk := $(lastword $(MAKEFILE_LIST))
include $(dir $(_common.mk))prelude.mk

#
# Variables

# Possible values of GOOS/GOARCH:
# https://golang.org/doc/install/source#environment
export GOOS   = $(call lazyonce,_GOOS,$(shell go env GOOS))
export GOARCH = $(call lazyonce,_GOARCH,$(shell go env GOARCH))

#
# User-facing targets

# To the extent reasonable, use target names that agree with the GNU
# standards.
#
# https://www.gnu.org/prep/standards/standards.html#Makefile-Conventions

all: build
.PHONY: all

build: ## (Common) Build the software
.PHONY: build

check: ## (Common) Check whether the software works; run the tests
.PHONY: check

lint: ## (Common) Perform static analysis of the software
.PHONY: lint

format: ## (Common) Apply automatic formatting+cleanup to source code
.PHONY: format

clean: ## (Common) Delete all files that are normally created by building the software
.PHONY: clean
# XXX: Rename this to maintainer-clean, per GNU?
clobber: ## (Common) Delete all files that this Makefile can re-generate
clobber: clean
.PHONY: clobber

#
# Targets: Default behavior

clean: _common_clean
_common_clean:
	rm -f test-suite.tap
.PHONY: _common_clean

check: lint build
	$(MAKE) -f $(firstword $(MAKEFILE_LIST)) test-suite.tap.summary
test-suite.tap: $(tools/tap-driver)
	@$(tools/tap-driver) cat $(sort $(filter %.tap,$^)) > $@

%.tap.summary: %.tap $(tools/tap-driver)
	@$(tools/tap-driver) summarize $<

%.tap: %.tap.gen $(tools/tap-driver) FORCE
	@{ $(abspath $<) || true; } 2>&1 | tee $@ | $(tools/tap-driver) stream -n $<
%.log: %.test FORCE
	@$(abspath $<) >$@ 2>&1; echo :exit-status: $$? >>$@
%.tap: %.log %.test $(tools/tap-driver)
	@{ \
		printf '%s\n' 'TAP version 13' '1..1' && \
		sed 's/^/#/' < $< && \
		sed -n '$${ s/^:exit-status: 0$$/ok 1/; s/^:exit-status: 77$$/ok 1 # SKIP/; s/^:exit-status: .*/not ok 1/; p; }' < $<; \
	} | tee $@ | $(tools/tap-driver) stream -n $*.test

endif
