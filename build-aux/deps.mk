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

$(OSS_HOME)/build-aux/pip-show.txt: docker/base-pip.docker.tag.local
	docker run --rm "$$(cat docker/base-pip.docker)" sh -c 'pip freeze --exclude-editable | cut -d= -f1 | xargs pip show' > $@
clean: build-aux/pip-show.txt.rm

$(OSS_HOME)/build-aux/go-version.txt: $(tools/write-ifchanged)
	echo $(_go-version) | $(tools/write-ifchanged) $@
clean: build-aux/go-version.txt.rm

$(OSS_HOME)/build-aux/py-version.txt: docker/base-python/Dockerfile
	{ grep -o 'python3=\S*' | cut -d= -f2; } < $< > $@
clean: build-aux/py-version.txt.rm

$(OSS_HOME)/build-aux/go1%.src.tar.gz:
	curl -o $@ --fail -L https://dl.google.com/go/$(@F)
build-aux/go.src.tar.gz.clean:
	rm -f build-aux/go1*.src.tar.gz
clobber: build-aux/go.src.tar.gz.clean
