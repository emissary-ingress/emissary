#!/usr/bin/env bats

load common

@test "prelude_str.mk: NL" {
	# Honestly, this checks `quote.shell` more than it does NL.
	check_expr_eq strict '$(NL)' $'\n'
}

@test "prelude_str.mk: SPACE" {
	# Honestly, this checks `quote.shell` more than it does SPACE.
	check_expr_eq strict '$(SPACE)' ' '
}
