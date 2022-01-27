#!/hint/bash

common_setup() {
	test_tmpdir="$(mktemp -d)"
	cp -a "$BATS_TEST_DIRNAME/.." "$test_tmpdir/build-aux"
	mkdir "$test_tmpdir/tools"
	cp -a "$BATS_TEST_DIRNAME/../../tools/src" "$test_tmpdir/tools/src"
	cd "$test_tmpdir"
	cat >Makefile <<-'__EOT__'
		.DEFAULT_GOAL = tst
		tst:
		.PHONY: tst
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

# Usage: make_expecting_go_error
#
# Run 'make', and expect an error message like:
#
#     build-aux/kubeapply.mk:16: *** This Makefile requires Go '1.11.4' or newer; you have '1.10.3'.  Stop.
make_expecting_go_error() {
	not make >& output
	cat output
	[[ "$(wc -l <output)" -eq 1 ]]
	[[ "$(cat output)" == *": *** This Makefile requires Go '1.11.4' or newer; you "*".  Stop." ]]
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
