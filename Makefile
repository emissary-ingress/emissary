.PHONY: run
run: build
	@echo " >>> running oauth server"
	./ambassador-oauth ] 

.PHONY: build
build: tools vendor
	@echo " >>> building"
	@go build ./...

.PHONY: clean
clean:
	@echo " >>> cleaning compiled objects and binaries"
	@go clean -tags netgo -i ./...

vendor:
	@echo " >>> installing dependencies"
	@dep ensure

format:
	@echo " >>> running format"
	go fmt ./...

tools:
	@command -v dep >/dev/null ; if [ $$? -ne 0 ]; then \
		echo " >>> installing go dep"; \
		curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh; \
	fi
