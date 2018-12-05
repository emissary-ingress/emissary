CLUSTER:=$(NAME).knaut

claim:	$(CLUSTER).clean $(CLUSTER)

shell:
	@exec env -u MAKELEVEL PS1="(dev) [\W]$$ " KUBECONFIG=$(CLUSTER) bash
.PHONY: shell
