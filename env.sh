#!/usr/bin/env bash

set -e
set -u

# This is the namespace you want to use for development in quay,
# probably your quay.io username:
export REGISTRY_NAMESPACE=datawire

# This is the external ip address of the ambassador you are using to
# route to the auth service under development:
export EXTERNAL_IP=35.230.52.23

# THese are set from your auth0 account:
export AUTH_CALLBACK_URL=http://35.230.52.23/callback
export AUTH_DOMAIN=rus123.auth0.com
export AUTH_AUDIENCE=https://rus123.auth0.com/api/v2/
export AUTH_CLIENT_ID=zTqxrmhGMqZ1J2TjML6QQTsh_kYqJrKv
export AUTH_CLIENT_SECRET=auN3fneKuu5HvEFU2wz_swLKBgje3mYbRJj45acXxzQRe_9FsfPuKRLKcDVbNH5H
