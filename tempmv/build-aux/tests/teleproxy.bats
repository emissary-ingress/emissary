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

@test "teleproxy.mk: TELEPROXY doesn't need sudo each time" {
	check_go_executable teleproxy.mk TELEPROXY
	if [[ "$TELEPROXY" == unsupported ]]; then
		skip
	fi

	# Ensure that the next invocation can't use `sudo`.
	mkdir bin
	cat >>bin/sudo <<-'__EOT__'
		#!/bin/sh
		false
	__EOT__
	chmod 755 bin/sudo
	PATH=$PWD/bin:$PATH

	make
}
