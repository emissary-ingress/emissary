# Configure GNU Make itself

SHELL = bash

.DEFAULT_GOAL = all

# Turn off .INTERMEDIATE file removal by marking all files as
# .SECONDARY.  .INTERMEDIATE file removal is a space-saving hack from
# a time when drives were small; on modern computers with plenty of
# storage, it causes nothing but headaches.
#
# https://news.ycombinator.com/item?id=16486331
.SECONDARY:

# If a recipe errors, remove the target it was building.  This
# prevents outdated/incomplete results of failed runs from tainting
# future runs.  The only reason .DELETE_ON_ERROR is off by default is
# for historical compatibility.
#
# If for some reason this behavior is not desired for a specific
# target, mark that target as .PRECIOUS.
.DELETE_ON_ERROR:
