PROFILE=PROD
ENV_FILE=e2e/env_test.sh

# This is the namespace you want to use for development in quay,
# probably your quay.io username:
REGISTRY_NAMESPACE=datawire

# This is the external ip address of the ambassador you are using to
# route to the auth service under development:
EXTERNAL_IP=35.232.120.106

# THese are set from your auth0 account:
AUTH_CALLBACK_URL=http://35.232.120.106/callback
AUTH_DOMAIN=rus123.auth0.com
AUTH_AUDIENCE=https://rus123.auth0.com/api/v2/
AUTH_CLIENT_ID=zTqxrmhGMqZ1J2TjML6QQTsh_kYqJrKv
AUTH_CLIENT_SECRET=auN3fneKuu5HvEFU2wz_swLKBgje3mYbRJj45acXxzQRe_9FsfPuKRLKcDVbNH5H
