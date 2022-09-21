#!/bin/bash
set -euE -o pipefail

# capture Go psuedo version
version="$(go run ./tools/src/goversion)"

# output the tag version so that it can be used by CI for things such as container scanning
echo "${version:1}"
