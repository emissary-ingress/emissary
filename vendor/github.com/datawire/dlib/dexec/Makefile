GOHOME ?= $(HOME)/src/github.com/golang/go
GOVERSION ?= 1.15.5
# This Makefile has utility targets to help when updating the
# borrowed_*.go files.

borrowed.patch: FORCE
	@for copy in $$(git ls-files ':*borrowed_*'); do \
	  orig=$${copy/borrowed_}; \
	  [[ $$orig == internal/* ]] || orig=os/exec/$$orig; \
	  echo diff -u $(GOHOME)/src/$$orig $$copy >&2; \
	  diff -u $(GOHOME)/src/$$orig $$copy || true; \
	done > $@

check-attribution:
	for copy in $$(git ls-files ':*borrowed_*'); do \
	  orig=$${copy/borrowed_}; \
	  [[ $$orig == internal/* ]] || orig=os/exec/$$orig; \
	  if grep -Fq "Go $(GOVERSION) $$orig" "$$copy" && grep -q 'Copyright .* The Go Authors' "$$copy"; then \
	    echo "$$copy : Looks OK"; \
	  else \
	    echo "$$copy : Doesn't claim copied from Go $(GOVERSION) $$orig or doesn't have the copyright statement"; \
	  fi; \
	done
.PHONY: check-attribution

check-MODIFIED: borrowed.patch
	< $< grep ^+ | grep -v MODIFIED | colordiff
.PHONY: check-MODIFIED

.PHONY: FORCE
.SECONDARY:
SHELL = bash
