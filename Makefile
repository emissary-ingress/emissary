AMBASSADOR_COMMIT = shared/edgy

# Git clone
# Ensure that GIT_DIR and GIT_WORK_TREE are unset so that `git bisect`
# and friends work properly.
define SETUP
	PS4=; set +x; { \
	    unset GIT_DIR GIT_WORK_TREE; \
	    if [ -e ambassador ]; then exit; fi ; \
	    set -x; \
	    git init ambassador; \
	    cd ambassador; \
	    if ! git remote get-url origin &>/dev/null; then \
	        git remote add origin https://github.com/datawire/ambassador; \
	        git remote set-url --push origin git@github.com:datawire/ambassador.git; \
	    fi; \
	    git fetch || true; \
	    if [ $(AMBASSADOR_COMMIT) != '-' ]; then \
	        git checkout $(AMBASSADOR_COMMIT); \
	    elif ! git rev-parse HEAD >/dev/null 2>&1; then \
	        git checkout origin/master; \
	    fi; \
	}
endef

DUMMY:=$(shell $(SETUP))

OSS_HOME ?= ambassador
include ${OSS_HOME}/Makefile
$(call module,apro,.)
