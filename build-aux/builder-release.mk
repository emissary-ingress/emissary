AWS_S3_BUCKET ?= datawire-static-files

export RELEASE_REGISTRY_ERR=$(RED)ERROR: please set the RELEASE_REGISTRY make/env variable to the docker registry\n       you would like to use for release$(END)

release/promote-oss/.main: $(tools/docker-promote)
	@[[ '$(PROMOTE_FROM_VERSION)' =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-.*)?$$ ]] || (echo >&2 'Must set PROMOTE_FROM_VERSION to a vSEMVER value'; exit 1)
	@[[ '$(PROMOTE_TO_VERSION)'   =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-.*)?$$ ]] || (echo >&2 'Must set PROMOTE_TO_VERSION to a vSEMVER value' ; exit 1)
	@[[ -n '$(PROMOTE_FROM_REPO)'                                     ]] || (echo >&2 'Must set PROMOTE_FROM_REPO' ; exit 1)
	@[[ -n '$(PROMOTE_TO_REPO)'                                       ]] || (echo >&2 'Must set PROMOTE_TO_REPO' ; exit 1)
	@case '$(PROMOTE_CHANNEL)' in \
	  ''|wip|early|test|hotfix) true;; \
	  *) echo >&2 'Unknown PROMOTE_CHANNEL $(PROMOTE_CHANNEL)'; exit 1;; \
	esac

	@printf "$(CYN)==> $(GRN)Promoting $(BLU)%s$(GRN) to $(BLU)%s$(GRN) (channel='$(BLU)%s$(GRN)')$(END)\n" '$(PROMOTE_FROM_VERSION)' '$(PROMOTE_TO_VERSION)' '$(PROMOTE_CHANNEL)'

	@printf '  pushing $(CYN)$(PROMOTE_TO_REPO):$(patsubst v%,%,$(PROMOTE_FROM_VERSION))$(END)...\n'
	$(tools/docker-promote) $(PROMOTE_FROM_REPO):$(patsubst v%,%,$(PROMOTE_FROM_VERSION)) $(PROMOTE_TO_REPO):$(patsubst v%,%,$(PROMOTE_TO_VERSION))
	docker push $(PROMOTE_TO_REPO):$(patsubst v%,%,$(PROMOTE_TO_VERSION))

ifneq ($(IS_PRIVATE),)
	@echo '$@: not pushing to S3 because this is a private repo'
else
	@printf '  pushing $(CYN)https://s3.amazonaws.com/$(AWS_S3_BUCKET)/emissary-ingress/$(PROMOTE_CHANNEL)stable.txt$(END)...\n'
	printf '%s' "$(patsubst v%,%,$(PROMOTE_TO_VERSION))" | aws s3 cp - s3://$(AWS_S3_BUCKET)/emissary-ingress/$(PROMOTE_CHANNEL)stable.txt

	@printf '  pushing $(CYN)s3://scout-datawire-io/emissary-ingress/$(PROMOTE_CHANNEL)app.json$(END)...\n'
	printf '{"application":"emissary","latest_version":"%s","notices":[]}' "$(patsubst v%,%,$(PROMOTE_TO_VERSION))" | aws s3 cp - s3://scout-datawire-io/emissary-ingress/$(PROMOTE_CHANNEL)app.json

	{ $(MAKE) \
	  push-manifests \
	  publish-docs-yaml; }
endif
.PHONY: release/promote-oss/.main

release/promote-oss/to-rc: $(tools/devversion)
	@test -n "$(RELEASE_REGISTRY)" || (printf "$${RELEASE_REGISTRY_ERR}\n"; exit 1)
	@[[ "$(VERSION)" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-rc\.[0-9]+|-dev)$$ ]] || (printf '$(RED)ERROR: VERSION=%s does not look like an RC or dev tag\n' "$(VERSION)"; exit 1)
	@set -e; { \
	  dev_version=$$($(tools/devversion)); \
	  printf "$(CYN)==> $(GRN)found version $(BLU)$$dev_version$(GRN).$(END)\n"; \
	  $(MAKE) release/promote-oss/.main \
	    PROMOTE_FROM_VERSION="$$dev_version" \
	    PROMOTE_TO_VERSION='$(VERSION)' \
	    PROMOTE_FROM_REPO='$(DEV_REGISTRY)/$(REPO)' \
	    PROMOTE_TO_REPO='$(RELEASE_REGISTRY)/$(REPO)' \
	    PROMOTE_CHANNEL='test'; \
	}
.PHONY: release/promote-oss/to-rc

# To be run from a checkout at the tag you are promoting _from_.
# This is normally run from CI by creating the GA tag.
release/promote-oss/to-ga: $(tools/devversion)
	@test -n "$(RELEASE_REGISTRY)" || (printf "$${RELEASE_REGISTRY_ERR}\n"; exit 1)
	@[[ "$(VERSION)" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-ea)?$$ ]] || (printf '$(RED)ERROR: VERSION=%s does not look like a GA tag\n' "$(VERSION)"; exit 1)
	@set -e; { \
	  dev_version=$$($(tools/devversion)); \
	  printf "$(CYN)==> $(GRN)found version $(BLU)$$dev_version$(GRN).$(END)\n"; \
	  $(MAKE) release/promote-oss/.main \
	    PROMOTE_FROM_VERSION="$$dev_version" \
	    PROMOTE_TO_VERSION='$(VERSION)' \
	    PROMOTE_FROM_REPO='$(DEV_REGISTRY)/$(REPO)' \
	    PROMOTE_TO_REPO='$(RELEASE_REGISTRY)/$(REPO)' \
	    PROMOTE_CHANNEL=''; \
	}
.PHONY: release/promote-oss/to-ga

# `make release/ga-check` is meant to be run by a human maintainer to
# check that CI did all the right things.
release/ga-check:
	{ $(OSS_HOME)/releng/release-ga-check \
	  --ga-version=$(patsubst v%,%,$(VERSION)) \
	  --chart-version=$(patsubst v%,%,$(CHART_VERSION)) \
	  --source-registry=$(RELEASE_REGISTRY) \
	  --image-name=$(LCNAME); }
