NAME=ambassador-ratelimit
PROFILE ?= dev

include bootstrap.mk

.PHONY: compile
compile: $(RATELIMIT_REPO)/vendor $(BIN)
	cd ${RATELIMIT_REPO}/src/service_cmd && go build -o $(RATELIMIT) ./
	cd ${RATELIMIT_REPO}/src/client_cmd && go build -o $(RATELIMIT_CLIENT) ./
	cd ${RATELIMIT_REPO}/src/config_check_cmd && go build -o $(RATELIMIT_CONFIG_CHECK) ./

# This is for managing minor diffs to upstream code. If we need
# anything more than minor diffs this probably won't work so well. We
# really don't want to have more than minor diffs though without a
# good reason.
.PHONY: diff
diff:
	cd ${RATELIMIT_REPO} && git diff > $(PATCH)

CLUSTER=rl-cluster.knaut
KUBEWAIT=$(BIN)/kubewait
TELEPROXY=$(BIN)/teleproxy

$(KUBEWAIT):
	go get github.com/datawire/teleproxy/cmd/kubewait

$(TELEPROXY):
	go get github.com/datawire/teleproxy/cmd/teleproxy
	sudo chown root $(TELEPROXY)
	sudo chmod u+s $(TELEPROXY)

export KUBECONFIG=${PWD}/$(CLUSTER)

include kubernaut.mk
include k8s.mk

.PHONY: claim
claim: $(CLUSTER).clean $(CLUSTER)

.PHONY: shell
shell: $(CLUSTER) $(TELEPROXY)
	@exec env -u MAKELEVEL PS1="(dev) [\W]$$ " PATH=$(PATH):$(BIN) bash

KUBE_URL=https://kubernetes/api/

.PHONY: proxy
proxy:
	curl -s teleproxy/api/shutdown || true
	@sleep 1
	$(TELEPROXY) > /tmp/teleproxy.log 2>&1 &
	@for i in 1 2 4 8 x; do \
		if [ "$$i" == "x" ]; then echo "ERROR: proxy did not come up"; exit 1; fi; \
		echo "Checking proxy: $(KUBE_URL)"; \
		if curl -sk $(KUBE_URL); then \
			echo -e "\n\nProxy UP!"; \
			break; \
		fi; \
		echo "Waiting $$i seconds..."; \
		sleep $$i; \
	done

.PHONY: clean
clean: $(CLUSTER).clean k8s.clean
	rm -rf $(BIN)

.PHONY: clobber
clobber: clean
	rm -rf go

.PHONY: run
run:
	USE_STATSD=false \
	REDIS_SOCKET_TYPE=tcp \
	REDIS_URL=ratelimit-redis:6379 \
	RUNTIME_ROOT=${PWD}/config RUNTIME_SUBDIRECTORY=. $(RATELIMIT)
