#!/hint/sh

AMBASSADOR_IMAGE=quay.io/datawire/ambassador:0.72.0

AMB_SIDECAR_IMAGE=$(cat docker/amb-sidecar-plugins.docker.knaut-push)
PROXY_IMAGE=$(cat docker/traffic-proxy.docker.knaut-push)
SIDECAR_IMAGE=$(cat docker/app-sidecar.docker.knaut-push)
CONSUL_CONNECT_INTEGRATION_IMAGE=$(cat docker/consul_connect_integration.docker.knaut-push)
MODEL_CLUSTER_APP_IMAGE=$(cat docker/model-cluster-app.docker.knaut-push)
MODEL_CLUSTER_GRPC_AUTH_IMAGE=$(cat docker/model-cluster-grpc-auth.docker.knaut-push)
MODEL_CLUSTER_HTTP_AUTH_IMAGE=$(cat docker/model-cluster-http-auth.docker.knaut-push)
MODEL_CLUSTER_LOAD_GRPC_AUTH_IMAGE=$(cat docker/model-cluster-load-grpc-auth.docker.knaut-push)
MODEL_CLUSTER_LOAD_HTTP_AUTH_IMAGE=$(cat docker/model-cluster-load-http-auth.docker.knaut-push)
MODEL_CLUSTER_UAA_IMAGE=$(cat docker/model-cluster-uaa.docker.knaut-push)
MAX_LOAD_IMAGE=$(cat docker/max-load.docker.knaut-push)
DEV_PORTAL_IMAGE=$(cat docker/dev-portal-server.docker.knaut-push)
INTERNAL_ACCESS_IMAGE=$(cat docker/apro-internal-access.docker.knaut-push)
OPENAPI_SERVER_IMAGE=$(cat docker/example-service.docker.knaut-push)

# 03-ambassador-pro-*.yaml
AMBASSADOR_LICENSE_KEY=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6ImRldiIsImV4cCI6NDcwMDgyNjEzM30.wCxi5ICR6C5iEz6WkKpurNItK3zER12VNhM8F1zGkA8

# 04-filter-oauth2-*.yaml
AUTH_TENANT_URL=https://ambassador.standalone.svc.cluster.local

# 04-filter-oauth2-auth0.yaml
# These come directly from https://manage.auth0.com/#/applications/DOzF9q7U2OrvB7QniW9ikczS1onJgyiC/settings
# The administrator credentials to see that page are at
#    $(keybase config get -b mountdir)/team/datawireio/secrets/auth0.auth0.apro-testing.*
_Auth0_Domain=ambassador-oauth-e2e.auth0.com
_Auth0_Client_ID=DOzF9q7U2OrvB7QniW9ikczS1onJgyiC
_Auth0_Client_Secret=MkpnAmzX-EEzV708qD_giNd9CF_R-owNau94QZVgOfna9FYf-SdTvATuNkrEDBk-
# Make sure that:
#  - "${AUTH_TENANT_URL}/callback" is in the "Allowed Callback URLs" textbox
#  - "${AUTH_TENANT_URL}" is in the "Allowed Web Origins" textbox
#  - A test user account is set up (and configured in ./tests/cluster/oauth-e2e/idp_auth0.js)
IDP_AUTH0_PROVIDER_URL=https://${_Auth0_Domain}
IDP_AUTH0_AUDIENCE=https://${_Auth0_Domain}/api/v2/
IDP_AUTH0_CLIENT_ID=${_Auth0_Client_ID}
IDP_AUTH0_CLIENT_SECRET=${_Auth0_Client_Secret}

# 04-filter-oauth2-keycloak.yaml
# Keycloak is configured statically in 04-keycloak.yaml
IDP_KEYCLOAK_PROVIDER_URL=http://keycloak.localdev.svc.cluster.local/auth/realms/apro
IDP_KEYCLOAK_AUDIENCE=app
IDP_KEYCLOAK_CLIENT_ID=app
IDP_KEYCLOAK_CLIENT_SECRET=8517c278-0ae8-40e5-b418-20199b7e3fb5

# 04-filter-oauth2-okta.yaml
# These come directly from https://dev-264701-admin.okta.com/admin/app/oidc_client/instance/0oaeshpr0wKNbyWQn356/#tab-general
# The administrator credentials to view that page are at
#    $(keybase config get -b mountdir)/team/datawireio/secrets/okta.dev-264701.firstname_lastname.*
_Okta_Org_URL=https://dev-264701.okta.com
_Okta_Client_ID=0oaeshpr0wKNbyWQn356
_Okta_Client_Secret=7Z-C1IIxDSzr1ICmZgnKt8G1_Mdtm2CpqvKSNnXd
# Make sure that:
#  - "${AUTH_TENANT_URL}/callback" is in the "Login redirect URIs" field
#  - A test user account is set up (and configured in ./tests/cluster/oauth-e2e/idp_okta.js)
IDP_OKTA_PROVIDER_URL=${_Okta_Org_URL}/oauth2/default
IDP_OKTA_AUDIENCE=api://default
IDP_OKTA_CLIENT_ID=${_Okta_Client_ID}
IDP_OKTA_CLIENT_SECRET=${_Okta_Client_Secret}

# 04-filter-oauth2-azure.yaml
# These come directly from https://portal.azure.com/#blade/Microsoft_AAD_RegisteredApps/ApplicationMenuBlade/Overview/appId/3e1ce1e1-094d-4ff5-baed-5ba99e32f809/isMSAApp/
_Azure_Client_ID=3e1ce1e1-094d-4ff5-baed-5ba99e32f809
_Azure_Tenant_ID=8538d7d0-9b03-40c9-b72f-378b44ed97e2
# This comes directly from https://portal.azure.com/#blade/Microsoft_AAD_RegisteredApps/ApplicationMenuBlade/Credentials/appId/3e1ce1e1-094d-4ff5-baed-5ba99e32f809/isMSAApp/
_Azure_Client_Secret='d@HzogM7321tAHXVhI]E]Lt@6.s=TEyb'
# Make sure that:
#  - "${AUTH_TENANT_URL}/callback" is added as a "Redirect URI" at https://portal.azure.com/#blade/Microsoft_AAD_RegisteredApps/ApplicationMenuBlade/Authentication/appId/3e1ce1e1-094d-4ff5-baed-5ba99e32f809/isMSAApp/
#  - A test user account is set up (and configured in ./tests/cluster/oauth-e2e/idp_azure.js)
# The administrator credentials to log in to portal.azure.com are at
#    $(keybase config get -b mountdir)/team/datawireio/secrets/azure.portal.apro-testing.*
IDP_AZURE_PROVIDER_URL=https://login.microsoftonline.com/${_Azure_Tenant_ID}/v2.0
IDP_AZURE_CLIENT_ID=${_Azure_Client_ID}
IDP_AZURE_CLIENT_SECRET=${_Azure_Client_Secret}

# 04-filter-oauth2-uaa.yaml, 04-uaa.yaml
# UAA is configured statically in 04-uaa.yaml
IDP_UAA_PROVIDER_URL=http://uaa.standalone.svc.cluster.local/uaa
IDP_UAA_CLIENT_ID=ambassador
IDP_UAA_CLIENT_SECRET=ambassador

# 04-filter-oauth2-google.yaml
# These come directly from https://console.developers.google.com/apis/credentials/oauthclient/863968914497-8u7d8610nhvnpdf52s3krtsqahlss5mv.apps.googleusercontent.com?project=datawireio&folder&organizationId=1023803081209
IDP_GOOGLE_PROVIDER_URL=https://accounts.google.com/
IDP_GOOGLE_CLIENT_ID=863968914497-8u7d8610nhvnpdf52s3krtsqahlss5mv.apps.googleusercontent.com
IDP_GOOGLE_CLIENT_SECRET=x1LhmCCGk_5AHs3iGlRiyvOV
