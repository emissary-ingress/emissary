#!/bin/sh
set -e

SCRIPT_DIR=$(dirname $0)

PKG_NAME=$(cat package.json | jq -r '.name + "@" + .version')
if [[ -z "${PKG_NAME}" ]]; then
  >&2 echo "ERROR: Could not get package name"
  exit 1
fi

npm install >&2
license-checker --excludePackages "${PKG_NAME}" --customPath "${SCRIPT_DIR}/customLicenseFormat.json" --json
