NAME            = ambassador-pro
DOCKER_REGISTRY = quay.io/datawire
K8S_IMAGES      = docker/ambassador-pro
K8S_DIR         = e2e-oauth/k8s
K8S_ENV         = e2e-oauth/env.sh

export CGO_ENABLED = 0

include build-aux/go-mod.mk
include build-aux/k8s.mk
include build-aux/teleproxy.mk
include build-aux/help.mk

.DEFAULT_GOAL = help

#
# Main

docker/ambassador-pro.docker: docker/ambassador-pro/ambassador-oauth
docker/ambassador-pro/ambassador-oauth: bin_linux_amd64/ambassador-oauth
	cp $< $@

clean:
	rm -f e2e-oauth/k8s/??-ambassador-certs.yaml e2e-oauth/k8s/*.pem

#
# Check

# Generate the TLS secret
%/cert.pem %/key.pem: | %
	openssl req -x509 -newkey rsa:4096 -keyout $*/key.pem -out $*/cert.pem -days 365 -nodes -subj "/C=US/ST=Florida/L=Miami/O=SomeCompany/OU=ITdepartment/CN=ambassador.datawire.svc.cluster.local"
e2e-oauth/k8s/02-ambassador-certs.yaml: e2e-oauth/k8s/cert.pem e2e-oauth/k8s/key.pem
	kubectl --namespace=datawire create secret tls --dry-run --output=yaml ambassador-certs --cert e2e-oauth/k8s/cert.pem --key e2e-oauth/k8s/key.pem > $@

deploy: e2e-oauth/k8s/02-ambassador-certs.yaml
apply: e2e-oauth/k8s/02-ambassador-certs.yaml

e2e-oauth/node_modules: e2e-oauth/package.json $(wildcard e2e-oauth/package-lock.json)
	cd $(@D) && npm install
	@test -d $@
	@touch $@

check-e2e: ## Check: e2e tests
check-e2e: e2e-oauth/node_modules deploy
	$(MAKE) proxy
	cd e2e-oauth && npm test
	$(MAKE) unproxy
.PHONY: check-e2e
check: check-e2e

#
# Utility targets

push-tagged-image: ## docker push
push-tagged-image: docker/ambassador-pro.docker.push

run: ## Run ambassador-oauth locally
run: bin_$(GOOS)_$(GOARCH)/ambassador-oauth
	./$<
.PHONY: run
