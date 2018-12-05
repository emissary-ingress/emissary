TELEPROXY=$(CURDIR)/teleproxy

TELEPROXY_VERSION=0.3.2
# This should maybe be replaced with a lighterweight dependency
GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)

$(TELEPROXY):
	curl -o $(TELEPROXY) https://s3.amazonaws.com/datawire-static-files/teleproxy/$(TELEPROXY_VERSION)/$(GOOS)/$(GOARCH)/teleproxy
	sudo chown root $(TELEPROXY)
	sudo chmod go-w,a+sx $(TELEPROXY)

KUBE_URL=https://kubernetes/api/

proxy: $(CLUSTER) unproxy
	@sleep 1
	KUBECONFIG=$(CLUSTER) $(TELEPROXY) > /tmp/teleproxy.log 2>&1 &
	@for i in 1 2 4 8 16 32 64 x; do \
		if [ "$$i" == "x" ]; then echo "ERROR: proxy did not come up"; exit 1; fi; \
		echo "Checking proxy: $(KUBE_URL)"; \
		if curl -sk $(KUBE_URL); then \
			echo -e "\n\nProxy UP!"; \
			break; \
		fi; \
		echo "Waiting $$i seconds..."; \
		sleep $$i; \
	done
.PHONY: proxy

unproxy: $(TELEPROXY)
	curl -s 127.254.254.254/api/shutdown || true
.PHONY: unproxy

proxy.clobber:
	rm -f $(TELEPROXY)
