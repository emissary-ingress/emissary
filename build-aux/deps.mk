# This file deals with Emissary's (non-tool) dependencies.
# (tool dependencies are in tools.mk)

go-mod-tidy:
.PHONY: go-mod-tidy

go-mod-tidy: go-mod-tidy/main
go-mod-tidy/main: $(OSS_HOME)/build-aux/go-version.txt
	rm -f go.sum
	GOFLAGS=-mod=mod go mod tidy -compat=$$(cut -d. -f1,2 < $<) -go=$$(cut -d. -f1,2 < $<)
.PHONY: go-mod-tidy/main

vendor: FORCE
	go mod vendor
clean: vendor.rm-r

# Note that we use `uv pip list`, but `pip3 show` -- this is because `uv pip
# show` doesn't include the license information, and we need that for our
# reports. We also have to be very explicit about which Python interpreter
# `uv` should use and which pip3 to use, because we do _not_ want system
# packages in here.
#
# Finally, even though pip-show.txt depends on having the virtualenv set
# up, the rule must always run (hence the FORCE).

$(OSS_HOME)/build-aux/pip-show.txt: FORCE $(OSS_HOME)/.venv
	echo "Generating $@ in $(OSS_HOME)/build-aux"
	uv pip list --python $(OSS_HOME)/.venv/bin/python --format=freeze --exclude-editable | cut -d= -f1 | xargs $(OSS_HOME)/.venv/bin/pip3 show | egrep '^([A-Za-z-]+: |---)' > $@
clean: build-aux/pip-show.txt.rm

$(OSS_HOME)/build-aux/go-version.txt: $(_go-version/deps)
	{ sed -nr 's/^go ([0-9]+([.][0-9]+)*)/\1/p' go.mod; } > $@
clean: build-aux/go-version.txt.rm

$(OSS_HOME)/build-aux/py-version.txt: pyproject.toml
	{ yq '.project.requires-python | capture("(?<version>3.[0-9]+(\.[0-9]+)?)").version' $(OSS_HOME)/pyproject.toml; } < $< > $@
clean: build-aux/py-version.txt.rm

# Make sure that it's e.g. 1.21.0, not just 1.21
$(OSS_HOME)/build-aux/go1%.src.tar.gz:
	@version="$*"; \
	echo "version is $$version"; \
	if echo "$$version" | grep -q '^\.[0-9][0-9]*$$'; then \
		version="$$version.0"; \
	fi; \
	echo "Pulling from https://go.dev/dl/go1$$version.src.tar.gz"; \
	curl -o $@ --fail -L "https://go.dev/dl/go1$$version.src.tar.gz"

build-aux/go.src.tar.gz.clean:
	rm -f build-aux/go1*.src.tar.gz
clobber: build-aux/go.src.tar.gz.clean
