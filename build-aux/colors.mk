ifeq ($(words $(filter $(abspath $(lastword $(MAKEFILE_LIST))),$(abspath $(MAKEFILE_LIST)))),1)

_colors.mk := $(lastword $(MAKEFILE_LIST))
include $(dir $(_colors.mk))prelude.mk

# CSI is the "control sequence introducer" that initiates a control
# seqnece.
_csi := $(shell printf '\033[')

# Usage: $(call _cs,INT_ARG1 INT_ARG2..., OPERATION)
#
# Evaluate to a "control sequence" for the terminal.
_cs = $(_csi)$(call joinlist,;,$1)$(strip $2)

# Usage: $(call _sgr, prop1=val1 prop2=val2)
#
# Evaluate to the SGR control sequence to change how text is rendered,
# using human-friendly key/value pairs.
#
# Unknown key/value pairs are ignored.
#
# Known settings:
#
#  `reset`       reset all properties
#
#  `wgt=bold`    font weight = bold
#  `wgt=normal`  font weight = normal (not bold)
#
#  `fg=blk`      foreground color = black
#  `fg=red`      foreground color = red
#  `fg=grn`      foreground color = green
#  `fg=yel`      foreground color = yellow
#  `fg=blu`      foreground color = blue
#  `fg=prp`      foreground color = purple
#  `fg=cyn`      foreground color = cyan
#  `fg=wht`      foreground color = white
#  `fg=def`      foreground color = default
_sgr = $(call _cs,$(foreach param,$1,$(_sgr/$(subst =,/,$(param)))),m)

# The definitions of those settings:
_sgr/reset  = 0
_sgr/wgt/bold   = 1
_sgr/wgt/normal = 22
_sgr/fg/blk = 30
_sgr/fg/red = 31
_sgr/fg/grn = 32
_sgr/fg/yel = 33
_sgr/fg/blu = 34
_sgr/fg/prp = 35
_sgr/fg/cyn = 36
_sgr/fg/wht = 37
# 38 is 8bit/24bit color
_sgr/fg/def = 39

# Now expose things for public consumption...
#
# Choose colors carefully. If they don't work on both a black
# background and a white background, pick other colors (so white,
# yellow, and black are poor choices).
RED = $(call _sgr,wgt=bold fg=red)
GRN = $(call _sgr,wgt=bold fg=grn)
BLU = $(call _sgr,wgt=bold fg=blu)
CYN = $(call _sgr,wgt=bold fg=cyn)
BLD = $(call _sgr,wgt=bold)
END = $(call _sgr,reset)

endif
