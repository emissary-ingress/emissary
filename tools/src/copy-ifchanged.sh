#!/usr/bin/env bash
# Copyright 2019 Datawire. All rights reserved.

if ! cmp -s "$1" "$2"; then
	cp -f "$1" "$2"
fi
