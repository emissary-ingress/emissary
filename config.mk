PROFILE ?= dev

# Default profile
REGISTRY = quay.io
NAMESPACE = datawire
REPO = $(NAMESPACE)/$(NAME)-$(PROFILE)
VERSION = $(HASH)
RATELIMIT_IMAGE = $(REGISTRY)/$(REPO):ratelimit-$(VERSION)
PROXY_IMAGE = $(REGISTRY)/$(REPO):proxy-$(VERSION)
SIDECAR_IMAGE = $(REGISTRY)/$(REPO):sidecar-$(VERSION)

ifeq ($(PROFILE),prod)
  REPO = $(NAMESPACE)/$(NAME)
endif

ifeq ($(PROFILE),dev)
  # no overrides
endif
