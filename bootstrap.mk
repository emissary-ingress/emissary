ifeq ("$(GOPATH)","")
export GOPATH=$(PWD)/go
endif

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

bootstrap: $(RATELIMIT_REPO)/vendor $(BIN)
