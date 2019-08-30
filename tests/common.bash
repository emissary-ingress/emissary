#!/hint/bash

common_setup() {
	test_tmpdir="$(mktemp -d)"
	ln -s "$BATS_TEST_DIRNAME/.." "$test_tmpdir/build-aux"
	cd "$test_tmpdir"
	cat >Makefile <<-'__EOT__'
		.DEFAULT_GOAL = all
		all:
		.PHONY: all
		include build-aux/prelude.mk
		expr-eq-strict-actual: FORCE; printf '%s' $(call quote.shell,$(EXPR)) > $@
		expr-eq-echo-actual: FORCE; echo $(EXPR) > $@
		expr-eq-sloppy-actual: FORCE; echo $(foreach w,$(EXPR),$w) > $@
	__EOT__
}
setup() { common_setup; }

common_teardown() {
	cd /
	rm -rf -- "$test_tmpdir"
}
teardown() { common_teardown; }

# Usage: check_executable SNIPPET.mk VARNAME
check_executable() {
	[[ $# = 2 ]]
	local snippet=$1
	local varname=$2

	cat >>Makefile <<-__EOT__
		include build-aux/${snippet}
		include build-aux/var.mk
		all: \$(${varname}) \$(var.)${varname}
	__EOT__

	make

	local varvalue
	varvalue="$(cat "build-aux/.var.${varname}")"

	[[ "$varvalue" == /* ]]
	[[ -f "$varvalue" && -x "$varvalue" ]]

	eval "${varname}=\$varvalue"
}

check_expr_eq() {
	[[ $# = 3 ]]
	local mode=$1
	local expr=$2
	local expected=$3

	case "$mode" in
		strict) printf '%s' "$expected" > expected;;
		echo) echo $expected > expected;;
		sloppy) echo $expected > expected;;
	esac

	make EXPR="$expr" "expr-eq-${mode}-actual"

	diff -u expected "expr-eq-${mode}-actual"
}

not() {
	# This isn't just "I find 'not' more readable than '!'", it
	# serves an actual purpose.  '!' won't trigger an errexit, so
	# it's no good for assertions.  However, it can affect the
	# return value of a function, and that function can trigger an
	# errexit.
	! "$@"
}
