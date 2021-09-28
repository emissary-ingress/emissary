#!/usr/bin/env bash
# Copyright 2019 Datawire. All rights reserved.

if cmp -s "$1" "$2"; then
	rm -f "$1" || :
else
	mv -f "$1" "$2"
fi
