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
