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
#  Boolean support:
#  - Variable: TRUE  = T
#  - Variable: FALSE =
#  - Function: not
#
#  String support:
#  - Variable: export NL
#  - Variable:        SPACE
#  - Variable:        COMMA
#  - Function: str.eq
#
#  Unsigned integer support:
#  - Function: uint.max
#  - Function: uint.min
#  - Function: uint.eq
#  - Function: uint.ge
#  - Function: uint.le
#  - Function: uint.gt
#  - Function: uint.lt
#
#  Path support:
#  - Function: path.trimprefix
#  - Function: path.addprefix
#
#  Build tool support:
#  - Variable: export GOHOSTOS
#  - Variable: export GOHOSTARCH
#  - Variable: build-aux.dir
#
#  Other support:
#  - Function: joinlist
#  - Function: quote.shell
#  - Function: lazyonce
#  - .PHONY Target: noop
#  - .PHONY Target: FORCE
#
#  Internal use:
#  - Variable: _prelude.go.VERSION      (exposed as go-mod.mk:go.goversion)
#  - Function: _prelude.go.VERSION.HAVE (exposed as go-mod.mk:go.goversion.HAVE)
#  - Variable: _prelude.go.ensure       (used by go-mod.mk)
#
## common.mk targets ##
#  - clobber
ifeq ($(words $(filter $(abspath $(lastword $(MAKEFILE_LIST))),$(abspath $(MAKEFILE_LIST)))),1)
_prelude.mk := $(lastword $(MAKEFILE_LIST))

# For my own sanity with organization, I've split out several "groups"
# of functionality from this file.  Maybe that's a sign that this has
# grown too complex.  Maybe we should stop fighting it and just use
# [GMSL](https://gmsl.sourceforge.io/).  Absolutely nothing in any of
# the `prelude_*.mk` files is allowed to be eager, so ordering doesn't
# matter.  Anything eager must go in this main `prelude.mk` file.
include $(dir $(_prelude.mk))prelude_bool.mk
include $(dir $(_prelude.mk))prelude_str.mk
include $(dir $(_prelude.mk))prelude_uint.mk
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
quote.shell = $(subst $(NL),'"$${NL}"','$(subst ','\'',$1)')

# Usage: VAR = $(call lazyonce,VAR,EXPR)
#
# Caches the value of EXPR (in case it's expensive/slow) once it is
# evaluated, but doesn't eager-evaluate it either.
lazyonce = $(eval $(strip $1) := $2)$2
_lazyonce.disabled = $(FALSE)

ifeq ($(MAKE_VERSION),3.81)
  define _lazyonce.print_warning
    $(warning The 'lazyonce' function is known to trigger a memory corruption bug in GNU Make 3.81)
    $(warning Disabling the 'lazyonce' function; upgrade your copy of GNU Make for faster builds)
    $(eval _lazyonce.need_warning = $(FALSE))
  endef
  _lazyonce.need_warning = $(TRUE)
  # The second $(if) is just so that the evaluated result output of
  # _lazyonce.print_warning isn't part of the returned value.
  lazyonce = $(if $(_lazyonce.need_warning),$(if $(_lazyonce.print_warning),))$2
  _lazyonce.disabled = $(TRUE)

  # These are use a lot, so go ahead and eager-evaluate them to speed
  # things up.
  _prelude.go.HAVE := $(_prelude.go.HAVE)
  _prelude.go.VERSION := $(_prelude.go.VERSION)
endif

#
# Variable constants

build-aux.dir = $(patsubst %/,%,$(dir $(_prelude.mk)))

#
# Targets

noop:
	@true
.PHONY: noop

# Sometimes we have a file-target that we want Make to always try to
# re-generate (such as compiling a Go program; we would like to let
# `go install` decide whether it is up-to-date or not, rather than
# trying to teach Make how to do that).  We could mark it as .PHONY,
# but that tells Make that "this isn't a real file that I expect to
# ever exist", which has a several implications for Make, most of
# which we don't want.  Instead, we can have them *depend* on a .PHONY
# target (which we'll name "FORCE"), so that they are always
# considered out-of-date by Make, but without being .PHONY themselves.
.PHONY: FORCE

endif
