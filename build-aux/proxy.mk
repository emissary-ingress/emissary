TELEPROXY=$(GOBIN)/teleproxy

$(TELEPROXY):
	$(GO) get github.com/datawire/teleproxy/cmd/teleproxy
	sudo chown root $(TELEPROXY)
	sudo chmod u+s $(TELEPROXY)

KUBE_URL=https://kubernetes/api/

proxy: $(TELEPROXY)
	curl -s 127.254.254.254/api/shutdown || true
	@sleep 1
	$(TELEPROXY) > /tmp/teleproxy.log 2>&1 &
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

proxy.clobber:
	rm teleproxy
