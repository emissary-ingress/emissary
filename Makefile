.SHELL: /bin/bash

.PHONY: run
run: install
	@echo " >>> running oauth server"
	ambassador-oauth 

.PHONY: install
install: tools vendor
	@echo " >>> building"
	@go install ./cmd/...

.PHONY: clean
clean:
	@echo " >>> cleaning compiled objects and binaries"
	@go clean -i ./...

.PHONY: test
test:
	@echo " >>> testing code.."
	@go test cmd/ambassador-oauth/*.go

vendor:
	@echo " >>> installing dependencies"
	@dep ensure -vendor-only

format:
	@echo " >>> running format"
	go fmt ./...

tools:
	@command -v dep >/dev/null ; if [ $$? -ne 0 ]; then \
		echo " >>> installing go dep"; \
		curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh; \
	fi

TEMPLATES=scripts/policy-crd.yaml scripts/authorization-srv.yaml scripts/httpbin.yaml scripts/httpbin-policy.yaml

MANIFESTS_DIR=manifests
MANIFESTS=$(TEMPLATES:scripts/%.yaml=$(MANIFESTS_DIR)/%.yaml)

$(MANIFESTS_DIR)/%.yaml : scripts/%.yaml env.sh
	mkdir -p $(MANIFESTS_DIR) && cat $< | /bin/bash -c "source env.sh && envsubst" > $@

.PHONY: deploy
deploy: $(MANIFESTS)
	@echo " >>> deploying"
	for FILE in $?; do kubectl apply -f $$FILE; done

.PHONY: clobber
clobber: clean
	rm -rf $(MANIFESTS_DIR)
