#!/usr/bin/env bats

load common

@test "teleproxy.mk: TELEPROXY" {
	check_go_executable teleproxy.mk TELEPROXY
	# TODO: Check that $TELEPROXY behaves correctly
}

@test "teleproxy.mk: TELEPROXY (CGO_ENABLED=0)" {
	echo 'export CGO_ENABLED = 0' >> Makefile
	check_go_executable teleproxy.mk TELEPROXY
	# TODO: Check that $TELEPROXY behaves correctly
}
