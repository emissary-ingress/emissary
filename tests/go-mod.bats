#!/usr/bin/env bats

load common

setup() {
	common_setup
	echo module testlocal > go.mod
}

@test "go-mod.mk: GOTEST2TAP" {
	check_go_executable go-mod.mk GOTEST2TAP
	# TODO: Check that $GOTEST2TAP behaves correctly
}

@test "go-mod.mk: GOLANGCI_LINT" {
	check_go_executable go-mod.mk GOLANGCI_LINT
	# TODO: Check that $GOLANGCI_LINT behaves correctly
}
