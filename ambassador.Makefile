PRO_HOME := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
include Makefile
$(call module,apro,$(PRO_HOME))
