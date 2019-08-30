# Copyright 2018-2019 Datawire. All rights reserved.
#
# Useful bits for writing Makefiles or Makefile snippets.
#
## Eager inputs ##
#  (none)
## Lazy inputs ##
#  (none)
## Outputs ##
#
#  String support:
#  - Variable: NL
#  - Variable: SPACE
#
#  Path support:
#  - Function: path.trimprefix
#  - Function: path.addprefix
#
#  Build tool support:
#  - Variable: export GOHOSTOS
#  - Variable: export GOHOSTARCH
#  - Variable: build-aux.dir
#  - Variable: build-aux.bindir
#  - Function: build-aux.bin-go.rule
#  - Executable: FLOCK           ?= $(CURDIR)/build-aux/bin/flock # or /usr/bin/flock
#  - Executable: WRITE_IFCHANGED ?= $(CURDIR)/build-aux/bin/write-ifchanged
#  - Executable: COPY_IFCHANGED  ?= $(CURDIR)/build-aux/bin/copy-ifchanged
#  - Executable: MOVE_IFCHANGED  ?= $(CURDIR)/build-aux/bin/move-ifchanged
#  - Executable: TAP_DRIVER      ?= $(CURDIR)/build-aux/bin/tap-driver
#
#  Other support:
#  - Function: joinlist
#  - Function: quote.shell
#  - Function: lazyonce
#  - .PHONY Target: FORCE
#
## common.mk targets ##
#  - clobber
#
# This file on its own does not introduce a dependency on `go`, but
# calling the `build-aux.bin-go.rule` function introduces a hard
# dependency Go 1.11.4+.
ifeq ($(words $(filter $(abspath $(lastword $(MAKEFILE_LIST))),$(abspath $(MAKEFILE_LIST)))),1)
_prelude.mk := $(lastword $(MAKEFILE_LIST))

# For my own sanity with organization, I've split out several "groups"
# of functionality from this file.  Maybe that's a sign that this has
# grown too complex.  Maybe we should stop fighting it and just use
# [GMSL](https://gmsl.sourceforge.io/).  Absolutely nothing in any of
# the `prelude_*.mk` files is allowed to be eager, so ordering doesn't
# matter.  Anything eager must go in this main `prelude.mk` file.
include $(dir $(_prelude.mk))prelude_str.mk
include $(dir $(_prelude.mk))prelude_path.mk
include $(dir $(_prelude.mk))prelude_go.mk

#
# Functions

# Usage: $(call joinlist,SEPARATOR,LIST)
# Example: $(call joinlist,/,foo bar baz) => foo/bar/baz
joinlist=$(if $(word 2,$2),$(firstword $2)$1$(call joinlist,$1,$(wordlist 2,$(words $2),$2)),$2)

# Usage: $(call quote.shell,STRING)
# Example: $(call quote.shell,some'string"with`special characters) => "some'string\"with\`special characters"
#
# Based on
# https://git.lukeshu.com/autothing/tree/build-aux/Makefile.once.head/00-quote.mk?id=9384e763b00774603208b3d44977ed0e6762a09a
# but modified to make newlines work with shells other than Bash.
quote.shell = "$$(printf '%s\n' $(subst $(NL),' ','$(subst ','\'',$1)'))"

# Usage: VAR = $(call lazyonce,VAR,EXPR)
#
# Caches the value of EXPR (in case it's expensive/slow) once it is
# evaluated, but doesn't eager-evaluate it either.
lazyonce = $(eval $(strip $1) := $2)$($(strip $1))

#
# Variable constants

build-aux.dir = $(patsubst %/,%,$(dir $(_prelude.mk)))
build-aux.bindir = $(abspath $(build-aux.dir)/bin)

#
# Executables
#
# Have this section toward the end, so that it can eagerly use stuff
# defined above.

FLOCK           ?= $(call lazyonce,FLOCK,$(or $(shell which flock 2>/dev/null),$(build-aux.bindir)/flock))
COPY_IFCHANGED  ?= $(build-aux.bindir)/copy-ifchanged
MOVE_IFCHANGED  ?= $(build-aux.bindir)/move-ifchanged
WRITE_IFCHANGED ?= $(build-aux.bindir)/write-ifchanged
TAP_DRIVER      ?= $(build-aux.bindir)/tap-driver

$(build-aux.bindir):
	mkdir $@

clobber: _clobber-prelude
_clobber-prelude:
	rm -rf $(build-aux.bindir)
.PHONY: _clobber-prelude

$(build-aux.bindir)/%: $(build-aux.dir)/bin-sh/%.sh | $(build-aux.bindir)
	install $< $@

# Usage: $(eval $(call build-aux.bin-go.rule,BINARY_NAME,GO_PACKAGE))
define build-aux.bin-go.rule
$$(build-aux.bindir)/.$(strip $1).stamp: $$(build-aux.bindir)/.%.stamp: $$(build-aux.dir)/bin-go/%/go.mod $$(_prelude.go.lock) FORCE | $$(build-aux.bindir)
	cd $$(<D) && GO111MODULE=on $$(_prelude.go.lock)go build -o $$(abspath $$@) $2
endef
$(build-aux.bindir)/%: $(build-aux.bindir)/.%.stamp $(COPY_IFCHANGED)
	$(COPY_IFCHANGED) $< $@

# bin/.flock.stamp doesn't use build-aux.bin-go.rule, because bootstrapping
$(build-aux.bindir)/.flock.stamp: $(build-aux.bindir)/.%.stamp: $(build-aux.dir)/bin-go/%/go.mod $(shell find $(build-aux.dir)/bin-go/flock) | $(build-aux.bindir)
	cd $(<D) && GO111MODULE=on go build -o $(abspath $@) github.com/datawire/build-aux/bin-go/flock

#
# Targets

# Sometimes we have a file-target that we want Make to always try to
# re-generate (such as compiling a Go program; we would like to let
# `go install` decide whether it is up-to-date or not, rather than
# trying to teach Make how to do that).  We could mark it as .PHONY,
# but that tells Make that "this isn't a "this isn't a real file that
# I expect to ever exist", which has a several implications for Make,
# most of which we don't want.  Instead, we can have them *depend* on
# a .PHONY target (which we'll name "FORCE"), so that they are always
# considered out-of-date by Make, but without being .PHONY themselves.
.PHONY: FORCE

endif
