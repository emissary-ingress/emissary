#!/bin/bash

# Expections
# - Exist as /opt/image-build/install.sh
# - Get called from post-install.sh
# - Run all scripts in /opt/image-build/installers/
# - Be run once as part of prod docker build
# - Be run repeatedly in the builder container
# See also: installers/README.md

set -e

cd /opt/image-build/installers
shopt -s nullglob
for installer in /opt/image-build/installers/*; do
    if [ -x "$installer" ]; then
        echo Installing $(basename "$installer")
        "$installer"
    fi
done
