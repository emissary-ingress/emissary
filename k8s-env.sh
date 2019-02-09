#!/hint/sh

AMB_SIDECAR_IMAGE=$(cat docker/amb-sidecar.docker.knaut-push)
PROXY_IMAGE=$(cat docker/traffic-proxy.docker.knaut-push)
SIDECAR_IMAGE=$(cat docker/app-sidecar.docker.knaut-push)
CONSUL_CONNECT_INTEGRATION_IMAGE=$(cat docker/consul_connect_integration.docker.knaut-push)

# acceptance_test.js
EXTERNAL_IP=ambassador.standalone.svc.cluster.local
TESTUSER_EMAIL=testuser@datawire.com
TESTUSER_PASSWORD=TestUser321

# These come directly from
# https://manage.auth0.com/#/applications/DOzF9q7U2OrvB7QniW9ikczS1onJgyiC/settings
_Auth0_Domain=ambassador-oauth-e2e.auth0.com
_Auth0_Client_ID=DOzF9q7U2OrvB7QniW9ikczS1onJgyiC
_Auth0_Client_Secret=MkpnAmzX-EEzV708qD_giNd9CF_R-owNau94QZVgOfna9FYf-SdTvATuNkrEDBk-
# Make sure that:
#  - "https://${EXTERNAL_IP}/callback" is in the "Allowed Callback URLs" textbox
#  - "https://${EXTERNAL_IP}" is in the "Allowed Web Origins" textbox
#  - The TESTUSER_EMAIL/TESTUSER_PASSWORD account is set up

# 03-ambassador-pro-*.yaml
AUTH_PROVIDER_URL=https://${_Auth0_Domain}
AMBASSADOR_LICENSE_KEY=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6ImRldiIsImV4cCI6NDcwMDgyNjEzM30.wCxi5ICR6C5iEz6WkKpurNItK3zER12VNhM8F1zGkA8

AUTH_TENANT_URL=https://${EXTERNAL_IP}

# 04-tenants.yaml
#AUTH_AUDIENCE=https://${_Auth0_Domain}/api/v2/
#AUTH_CLIENT_ID=${_Auth0_Client_ID}
#AUTH_CLIENT_SECRET=${_Auth0_Client_Secret}

# --- Keycloak ---
AUTH_PROVIDER_URL=http://keycloak.standalone.svc.cluster.local/auth/realms/apro
AUTH_AUDIENCE=app
AUTH_CLIENT_ID=app
AUTH_CLIENT_SECRET=8517c278-0ae8-40e5-b418-20199b7e3fb5