#!/hint/sh

# 03-ambassador-pro-oauth.yaml
IMAGE=$(cat docker/ambassador-pro.docker.knaut-push)
AUTH_PROVIDER_URL=https://ambassador-oauth-e2e.auth0.com

# 04-tenants.yaml
AUTH_TENANT_URL=https://ambassador.datawire.svc.cluster.local
AUTH_AUDIENCE=https://ambassador-oauth-e2e.auth0.com/api/v2/
AUTH_CLIENT_ID=DOzF9q7U2OrvB7QniW9ikczS1onJgyiC
AUTH_CLIENT_SECRET=MkpnAmzX-EEzV708qD_giNd9CF_R-owNau94QZVgOfna9FYf-SdTvATuNkrEDBk-

# acceptance_test.js
EXTERNAL_IP=ambassador.datawire.svc.cluster.local
TESTUSER_EMAIL=testuser@datawire.com
TESTUSER_PASSWORD=TestUser321

# Unused?
AUTH_CALLBACK_URL=https://ambassador.datawire.svc.cluster.local/callback
AUTH_SCOPE=email
