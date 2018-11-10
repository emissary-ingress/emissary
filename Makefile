
ifeq ("$(GOPATH)","")
export GOPATH=$(PWD)/go
endif

include kubernaut.mk

RATELIMIT_REPO=$(GOPATH)/src/github.com/lyft/ratelimit
RATELIMIT_VERSION=v1.3.0

BIN=$(GOPATH)/bin

RATELIMIT=$(BIN)/ratelimit
RATELIMIT_CLIENT=$(BIN)/ratelimit_client
RATELIMIT_CONFIG_CHECK=$(BIN)/ratelimit_config_check

PATCH=$(PWD)/ratelimit.patch

$(RATELIMIT_REPO):
	mkdir -p $(RATELIMIT_REPO) && git clone https://github.com/lyft/ratelimit $(RATELIMIT_REPO)
	cd $(RATELIMIT_REPO) && git checkout -q $(RATELIMIT_VERSION)
	cd $(RATELIMIT_REPO) && git apply $(PATCH)

$(RATELIMIT_REPO)/vendor: $(RATELIMIT_REPO)
	cd $(RATELIMIT_REPO) && glide install

$(BIN):
	mkdir -p $(BIN)

.PHONY: compile
compile: $(RATELIMIT_REPO)/vendor $(BIN)
	cd ${RATELIMIT_REPO}/src/service_cmd && go build -o $(RATELIMIT) ./
	cd ${RATELIMIT_REPO}/src/client_cmd && go build -o $(BIN)/ratelimit_client ./
	cd ${RATELIMIT_REPO}/src/config_check_cmd && go build -o $(BIN)/ratelimit_config_check ./

# This is for managing minor diffs to upstream code. If we need
# anything more than minor diffs this probably won't work so well. We
# really don't want to have more than minor diffs though without a
# good reason.
.PHONY: diff
diff:
	cd ${RATELIMIT_REPO} && git diff > $(PATCH)

CLUSTER=rl-cluster.knaut

export KUBECONFIG=${PWD}/$(CLUSTER)


.PHONY: shell
shell: $(CLUSTER)
	@exec env -u MAKELEVEL PS1="(dev) [\W]$$ " PATH=$(PATH):$(BIN) bash

KUBEWAIT=$(BIN)/kubewait

$(KUBEWAIT):
	go get github.com/datawire/teleproxy/cmd/kubewait

TELEPROXY=$(BIN)/teleproxy

$(TELEPROXY):
	go get github.com/datawire/teleproxy/cmd/teleproxy
	sudo chown root $(TELEPROXY)
	sudo chmod u+s $(TELEPROXY)

.PHONY: tools
tools: $(TELEPROXY)

manifests: $(CLUSTER) $(KUBEWAIT)
	kubectl apply -f k8s
	$(KUBEWAIT) -f k8s

.PHONY: clean
clean: $(CLUSTER).clean
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
