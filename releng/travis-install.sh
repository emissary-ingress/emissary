#!/usr/bin/env bash
set -o errexit
set -o nounset
set -o pipefail

source releng/vars.sh

releng/install-kubectl.sh

pip install -r dev-requirements.txt
pip install -r ambassador/requirements.txt
npm install gitbook-cli netlify-cli