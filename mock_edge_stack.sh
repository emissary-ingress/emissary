#!/bin/bash

export SCOUT_DISABLE=1
export AMBASSADOR_CONFIG_BASE_DIR=${1:-/tmp/ambassador}
export AMBASSADOR_NAMESPACE=ambassador

INIT_CONFIG="${AMBASSADOR_CONFIG_BASE_DIR}/init-config"

if [ -d /ambassador/init-config ]; then
	rm -rf "${INIT_CONFIG}"
	mkdir -p "${INIT_CONFIG}"
	cp -pr /ambassador/init-config "${INIT_CONFIG}"
fi

bash /buildroot/ambassador/python/entrypoint.sh
