# This is the branch from ambassador-docs.git to pull for "make pull-docs".
# Override if you need to.
PULL_BRANCH ?= master

# This is the branch from ambassador-docs.git to push to for "make push-docs".
# Override if you need to.
PUSH_BRANCH ?= rel/v$(shell echo "$(RELEASE_VERSION)")

# ------------------------------------------------------------------------------
# Website
# ------------------------------------------------------------------------------

subtree-preflight:
	@if ! grep -q split_list_relevant_parents $$(PATH=$$(git --exec-path):$$PATH which git-subtree 2>/dev/null) /dev/null; then \
	    printf '$(RED)Please upgrade your git-subtree:$(END)\n'; \
	    printf '$(BLD)  sudo curl -fL https://raw.githubusercontent.com/LukeShu/git/lukeshu/subtree-2020-01-03/contrib/subtree/git-subtree.sh -o $$(git --exec-path)/git-subtree && sudo chmod 755 $$(git --exec-path)/git-subtree$(END)\n'; \
	    false; \
	fi
.PHONY: subtree-preflight

pull-docs: ## Pull the docs from ambassador-docs.git
pull-docs: subtree-preflight
	git subtree pull --prefix=docs https://github.com/datawire/ambassador-docs $(PULL_BRANCH)
.PHONY: pull-docs

push-docs: ## Push the docs to ambassador-docs.git
push-docs: subtree-preflight
# 1. As long as both (1) ${subtree_dir}/${subtree_dir}/ exists (i.e. the
#   'docs/' subtree contains a 'docs/' directory), and (2) there
#   subtree-merges in the relevant history that don't have 'git-subtree-dir:'
#   annotations (with current (2020-01-03) versions of git subtree that's all
#   non-rejoin merges), then:
#
#   You need to specify --onto to get it do to the right thing.  That's not an
#   implementation bug, it's an inherent consequence of the design of git
#   subtree.
#
# 2. Yes, fetch $(PULL_BRANCH) instead of $(PUSH_BRANCH); that way you can
#    `make push-docs PUSH_BRANCH=my-new-branch-that-does-not-yet-exist` and it
#    will do the right thing.
	@PS4=; set -ex; { \
	    git fetch https://github.com/datawire/ambassador-docs $(PULL_BRANCH); \
	    git subtree push --prefix=docs --rejoin --onto=$$(git rev-parse FETCH_HEAD) $(if $(GH_TOKEN),https://d6e-automaton:${GH_TOKEN}@github.com/,git@github.com:)datawire/ambassador-docs.git $(PUSH_BRANCH); \
	}
.PHONY: push-docs
