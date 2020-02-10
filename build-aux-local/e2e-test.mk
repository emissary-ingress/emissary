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
# Delete many things left over from the KAT tests.  This isn't meant
# to be hygenic; it's just that both the full KAT deployments and the
# full e2e deployments don't both fit in a Kubernaut cluster at the
# same time.
	kubectl --kubeconfig=$(DEV_KUBECONFIG) delete namespaces alt-namespace evil-namespace other-namespace plain-namespace same-ingress-1 same-ingress-2 same-mapping-1 same-mapping-2 secret-namespace-ingress tcp-namespace watt-rapid || true
	$(MAKE) deploy
	$(MAKE) -f $(SOURCE_apro)/build-aux-local/Makefile.e2e apply
	$(MAKE) -f $(SOURCE_apro)/build-aux-local/Makefile.e2e proxy
	$(MAKE) -f $(SOURCE_apro)/build-aux-local/Makefile.e2e check
.PHONY: e2etest-only

pytest-only: _e2etest-cleanup
_e2etest-cleanup:
# test_docker.py and a few others get super-duper unhappy if teleproxy
# is running.  IDK why... (it it spams the same line repeatedly to
# stderr so many thousands of times that my terminal's scrollback
# loses the "why")
	$(MAKE) -f $(SOURCE_apro)/build-aux-local/Makefile.e2e unproxy || true
# Delete the big things left over from the e2e tests.  This isn't
# meant to be hygenic; it's just that both the full KAT deployments
# and the full e2e deployments don't both fit in a Kubernaut cluster
# at the same time.
	kubectl --kubeconfig=$(DEV_KUBECONFIG) delete deployments uaa keycloak || true
.PHONY: _e2etest-cleanup

define _help.e2e-targets
  $(BLD)make $(BLU)e2etest$(END)             -- runs just the Go e2e tests.
endef
_help.targets += $(NL)$(NL)$(_help.e2e-targets)
