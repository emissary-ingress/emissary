# This is the branch from ambassador-docs.git to pull for "make pull-docs".
# Override if you need to.
PULL_BRANCH ?= master

# ------------------------------------------------------------------------------
# Website
# ------------------------------------------------------------------------------

pull-docs: ## Pull the docs from ambassador-docs.git
	@PS4=; set -ex; { \
	    git fetch https://github.com/datawire/ambassador-docs $(PULL_BRANCH); \
	    docs_head=$$(git rev-parse FETCH_HEAD); \
	    git subtree merge --prefix=docs "$${docs_head}"; \
	    git subtree split --prefix=docs --rejoin --onto="$${docs_head}"; \
	}
push-docs: ## Push the docs to ambassador-docs.git
	@PS4=; set -ex; { \
	    git fetch https://github.com/datawire/ambassador-docs master; \
	    docs_old=$$(git rev-parse FETCH_HEAD); \
	    docs_new=$$(git subtree split --prefix=docs --rejoin --onto="$${docs_old}"); \
	    git push $(if $(GH_TOKEN),https://d6e-automaton:${GH_TOKEN}@github.com/,git@github.com:)datawire/ambassador-docs.git "$${docs_new}:refs/heads/$(or $(PUSH_BRANCH),master)"; \
	}
.PHONY: pull-docs push-docs
