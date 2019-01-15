NAME            = ambassador-pro
DOCKER_REGISTRY = quay.io/datawire
K8S_IMAGES      = docker/ambassador-pro
K8S_DIR         = scripts
K8S_ENV         = env.sh

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
	rm -f key.pem cert.pem scripts/??-ambassador-certs.yaml

#
# Check

# Generate the TLS secret
%/cert.pem %/key.pem: | %
	openssl req -x509 -newkey rsa:4096 -keyout $*/key.pem -out $*/cert.pem -days 365 -nodes -subj "/C=US/ST=Florida/L=Miami/O=SomeCompany/OU=ITdepartment/CN=ambassador.datawire.svc.cluster.local"
scripts/02-ambassador-certs.yaml: scripts/cert.pem scripts/key.pem
	kubectl --namespace=datawire create secret tls --dry-run --output=yaml ambassador-certs --cert scripts/cert.pem --key scripts/key.pem > $@

deploy: scripts/02-ambassador-certs.yaml
apply: scripts/02-ambassador-certs.yaml

e2e/node_modules: e2e/package.json $(wildcard e2e/package-lock.json)
	cd $(@D) && npm install
	@test -d $@
	@touch $@

check-e2e: ## Check: e2e tests
check-e2e: e2e/node_modules deploy
	$(MAKE) proxy
	cd e2e && npm test
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
