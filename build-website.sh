#!/bin/bash
set -euo pipefail
IFS=$'\n\t'
# http://redsymbol.net/articles/unofficial-bash-strict-mode/
# because I don't know what I'm doing in Bash

set -x

# Build the documentation as usual
cd "$(dirname "$0")"
npm install
npm run build

# Remove the data-path attributed of every list item linking to index.html,
# which are the ones marked with data-level="1.1". This causes the GitBook
# scripts to redirect to the index page rather fetching and replacing just
# the content area, as they do for proper GitBook-generated pages.
sed -i.maclinuxincompat 's,<li class="chapter " data-level="1.1" data-path="[^"]*">,<li class="chapter " data-level="1.1">,' $(fgrep -rl 'data-level="1.1"' _book)
find . -type f -name "*.maclinuxincompat" -print0 | xargs -0 rm -f

# Replace index.html with our hand-crafted landing page
cp index.html _book/

# Copy YAML into _book/ as well.
cp -prv yaml _book/

