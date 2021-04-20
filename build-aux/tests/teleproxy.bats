#!/usr/bin/env bats

load common

@test "teleproxy.mk: tools/telepresence" {
	check_go_executable teleproxy.mk tools/telepresence
	# TODO: Check that $(tools/telepresence) behaves correctly
}

@test "teleproxy.mk: tools/telepresence (CGO_ENABLED=0)" {
	echo 'export CGO_ENABLED = 0' >> Makefile
	check_go_executable teleproxy.mk tools/telepresence
	# TODO: Check that $(tools/telepresence) behaves correctly
}
