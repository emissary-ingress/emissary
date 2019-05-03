#!/bin/sh

set -ex

curl -X POST -d @routes.json teleproxy/api/tables/
