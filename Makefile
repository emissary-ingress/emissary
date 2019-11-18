NAME ?= ambassador

OSS_HOME:=$(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))

include $(OSS_HOME)/builder/builder.mk

$(call module,ambassador,$(OSS_HOME))

include $(OSS_HOME)/build-aux-local/generate.mk

lint-oss:
	@PS4=; set -ex; { \
	  r=0; \
	  mypy --follow-imports=skip --ignore-missing-imports $(OSS_HOME)/python || r=$$?; \
	  (cd $(OSS_HOME) && golangci-lint run ./...) || r=$$?; \
	  shellcheck $(OSS_HOME)/*/*.sh $(OSS_HOME)/docker/*/*.sh $(OSS_HOME)/cmd/*/*.sh $(OSS_HOME)/builder/builder_bash_rc || r=$$?; \
	  exit $$r; \
	}
.PHONY: lint-oss

# Configure GNU Make itself
.SECONDARY:
.DELETE_ON_ERROR:
.PHONY: FORCE
