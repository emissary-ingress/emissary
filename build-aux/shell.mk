CLUSTER:=$(NAME).knaut

claim:	release $(CLUSTER)

release: $(CLUSTER).clean
.PHONY: release

shell:
	@exec env -u MAKELEVEL PS1="(dev) [\W]$$ " KUBECONFIG=$(CLUSTER) bash
.PHONY: shell
