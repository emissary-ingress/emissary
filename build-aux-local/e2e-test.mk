# Hook in to the build.mk targets
test-ready: e2etest-push
test: e2etest

e2etest-images: images
	@test -n "$(DEV_REGISTRY)" || (printf "$${REGISTRY_ERR}\n"; exit 1)
	@printf "$(CYN)==> $(GRN)Building $(BLU)e2e$(GRN) test images$(END)\n"
	$(MAKE) -f $(SOURCE_apro)/build-aux-local/Makefile.e2e images
.PHONY: e2etest-images

e2etest-push: e2etest-images
	@printf "$(CYN)==> $(GRN)Pushing $(BLU)e2e$(GRN) test images$(END)\n"
	$(MAKE) -f $(SOURCE_apro)/build-aux-local/Makefile.e2e push
.PHONY: e2etest-push

e2etest: test-ready
	$(MAKE) e2etest-only
.PHONY: e2etest

e2etest-only:
	@printf "$(CYN)==> $(GRN)Running $(BLU)e2e$(GRN) tests$(END)\n"
	$(MAKE) deploy
	$(MAKE) -f $(SOURCE_apro)/build-aux-local/Makefile.e2e apply
	$(MAKE) -f $(SOURCE_apro)/build-aux-local/Makefile.e2e proxy
	$(MAKE) -f $(SOURCE_apro)/build-aux-local/Makefile.e2e -j1 check
.PHONY: e2etest-only
