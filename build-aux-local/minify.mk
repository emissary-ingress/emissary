# These variables are useful to override if you want to test this
# Makefile outside of the builder image.
SRC_ROOT=/buildroot/apro
INSTALL_ROOT=/ambassador

ROLLUP_CONFIG_JS=$(SRC_ROOT)/cmd/amb-sidecar/webui/bindata/rollup.config.js

SRC_FILES:=$(filter-out %/rollup.config.js,$(shell find $(SRC_ROOT)/cmd/amb-sidecar/webui/bindata/ -type f -not -path $(SRC_ROOT)/cmd/amb-sidecar/webui/bindata/edge_stack/vendor/\*))

SRC_FILES_JS:=$(filter %.js,$(SRC_FILES))
SRC_FILES_OTHER:=$(filter-out $(SRC_FILES_JS),$(SRC_FILES))

INSTALLED_JS=$(SRC_FILES_JS:$(SRC_ROOT)/cmd/amb-sidecar/%=$(INSTALL_ROOT)/%)
INSTALLED_OTHER=$(SRC_FILES_OTHER:$(SRC_ROOT)/cmd/amb-sidecar/%=$(INSTALL_ROOT)/%)
INSTALLED=$(INSTALLED_JS) $(INSTALLED_OTHER)

$(INSTALL_ROOT)/%.js: $(SRC_ROOT)/cmd/amb-sidecar/%.js $(ROLLUP_CONFIG_JS)
	NODE_PATH="$$(npm root -g)" rollup -c $(ROLLUP_CONFIG_JS) -i $< -o $@

$(INSTALL_ROOT)/%: $(SRC_ROOT)/cmd/amb-sidecar/%
	install -D $< $@

all: $(INSTALLED)
	rm -f -- $(filter-out $(INSTALLED),$(shell find $(INSTALL_ROOT)/webui/bindata -type f))
