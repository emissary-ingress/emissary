
h2c_bundle.go: ../.circleci/bundle FORCE
	set -eux; { \
	  tmpdir=$$(mktemp -t -d); \
	  trap "rm -rf -- $$tmpdir" EXIT; \
	  cd $$tmpdir; \
	  go mod init dhttp; \
	  go mod edit -replace=golang.org/x/net=github.com/datawire/golang-x-net@821b2070f7eebf900bb2e45ec92d80699f99e80d; \
	  echo package dhttp > pkg.go; \
	  $(abspath $<) golang.org/x/net/http2/h2c > $(abspath $@); \
	}

../.circleci/bundle: FORCE
	$(MAKE) -C .. .circleci/bundle

.PHONY: FORCE
.DELETE_ON_ERROR:
.SECONDARY:
