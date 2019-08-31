#!/usr/bin/env bats

load common

@test "prelude_go.mk: GOHOSTOS" {
	[[ -n "$build_aux_expeced_GOHOSTOS" ]] || skip
	check_expr_eq strict '$(GOHOSTOS)' "$build_aux_expeced_GOHOSTOS"
}

@test "prelude_go.mk: GOHOSTARCH" {
	[[ -n "$build_aux_expeced_GOHOSTARCH" ]] || skip
	check_expr_eq strict '$(GOHOSTARCH)' "$build_aux_expeced_GOHOSTARCH"
}

@test "prelude_go.mk: _prelude.go.lock" {
	# TODO
}
