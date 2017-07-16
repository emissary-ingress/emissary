#!/bin/bash
set -euo pipefail
IFS=$'\n\t'
# http://redsymbol.net/articles/unofficial-bash-strict-mode/
# because I don't know what I'm doing in Bash

set -x

cd "$(dirname "$0")"
npm install
npm run build
sed -i "" 's,<li class="chapter " data-level="1.1" data-path="[^"]*">,<li class="chapter " data-level="1.1">,' $(fgrep -rl 'data-level="1.1"' _book)
cp index.html _book/
