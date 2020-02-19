# These variables are useful to override if you want to test this
# Makefile outside of the builder image.
SRC_ROOT=/buildroot/apro
INSTALL_ROOT=/ambassador

ROLLUP_CONFIG_JS=$(SRC_ROOT)/cmd/amb-sidecar/webui/bindata/rollup.config.js

SRC_FILES:=$(filter-out %/rollup.config.js,$(shell find $(SRC_ROOT)/cmd/amb-sidecar/webui/bindata/ -type f))
INSTALLED=$(SRC_FILES:$(SRC_ROOT)/cmd/amb-sidecar/%=$(INSTALL_ROOT)/%)

$(INSTALL_ROOT)/webui/bindata/edge_stack/vendor/%: $(SRC_ROOT)/cmd/amb-sidecar/webui/bindata/edge_stack/vendor/%
	install -m644 -D $< $@

$(INSTALL_ROOT)/%.js: $(SRC_ROOT)/cmd/amb-sidecar/%.js $(ROLLUP_CONFIG_JS)
	NODE_PATH="$$(npm root -g)" rollup -c $(ROLLUP_CONFIG_JS) -i $< -o $@

$(INSTALL_ROOT)/%: $(SRC_ROOT)/cmd/amb-sidecar/%
	install -m644 -D $< $@

all: $(INSTALLED)
	rm -f -- $(filter-out $(INSTALLED),$(shell find $(INSTALL_ROOT)/webui/bindata -type f))
