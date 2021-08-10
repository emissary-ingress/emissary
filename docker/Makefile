# Don't overthink things here. This Makefile just builds and pushes the
# test services, all with the same version (which must by in sync with the
# versions in the sources!).

VERSION=0.0.4

all: \
	test-auth-image \
	test-ratelimit-image \
	test-shadow-image \
	test-stats-image
.PHONY: all

test-auth-image:
	docker build -t docker.io/datawire/test_services:test-auth-$(VERSION) test-auth
	docker push docker.io/datawire/test_services:test-auth-$(VERSION)
	docker build --build-arg TLS="--tls" -t docker.io/datawire/test_services:test-auth-tls-$(VERSION) test-auth
	docker push docker.io/datawire/test_services:test-auth-tls-$(VERSION)
.PHONY: test-auth-image

test-ratelimit-image:
	docker build -t docker.io/datawire/test_services:test-ratelimit-$(VERSION) test-ratelimit
	docker push docker.io/datawire/test_services:test-ratelimit-$(VERSION)
.PHONY: test-ratelimit-image

test-shadow-image:
	docker build -t docker.io/datawire/test_services:test-shadow-$(VERSION) test-shadow
	docker push docker.io/datawire/test_services:test-shadow-$(VERSION)
.PHONY: test-shadow-image

test-stats-image:
	docker build -t docker.io/datawire/test_services:test-stats-$(VERSION) test-stats
	docker push docker.io/datawire/test_services:test-stats-$(VERSION)
.PHONY: test-stats-image
