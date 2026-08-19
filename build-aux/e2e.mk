include $(dir $(lastword $(MAKEFILE_LIST)))tools.mk

# End-to-end tests against a local k3d cluster.
#
# Prerequisites:
#   - Python venv active (`source .venv/bin/activate`) OR run via `uv run make ...`
#     so that Makefile invocations of `python3` resolve the project's deps.
#   - For dirty working trees, set VERSION to a short override (default below)
#     because `goversion` returns a string that exceeds Kubernetes' 63-char
#     label limit when composed into chart template labels.

E2E_CLUSTER       ?= emissary-e2e
E2E_NAMESPACE     ?= emissary
E2E_CRD_NAMESPACE ?= emissary-system
E2E_GATEWAY_URL   ?= http://localhost
E2E_COMPONENTS    ?= apiext emissary kat-client kat-server test-auth test-shadow test-stats

# Slots (isolated Emissary installs for tests that set global config) are
# defined in test/e2e/slots.sh, which CI calls too so the list and its port
# math have one definition. See the header of that file.
E2E_SLOTS_SH = $(OSS_HOME)/test/e2e/slots.sh

# k3d fixes published ports at cluster-creation time, so reserve the whole
# block up front. 8100-8159 leaves room for six slots; adding a seventh
# means recreating the cluster.
E2E_SLOT_PORT_RANGE ?= 8100-8159

E2E_IMAGE_REPO ?= ghcr.io/emissary-ingress/emissary

# Path to the version file goreleaser writes during `make images`. The tag
# goreleaser stamps onto its --snapshot builds follows its own versioning
# (not the Makefile's VERSION variable), so we read the authoritative value
# from here -- same approach as build-images.yaml in CI.
E2E_VERSION_FILE = $(OSS_HOME)/python/ambassador.version

.PHONY: e2e/cluster-up
e2e/cluster-up: $(tools/k3d) $(tools/kubectl)
	@if $(tools/k3d) cluster list -o json | grep -q '"name":"$(E2E_CLUSTER)"'; then \
	    echo "k3d cluster '$(E2E_CLUSTER)' already exists"; \
	else \
	    $(tools/k3d) cluster create $(E2E_CLUSTER) \
	        --api-port 6443 \
	        --port "80:80@loadbalancer" \
	        --port "443:443@loadbalancer" \
	        --port "6789:6789@loadbalancer" \
	        --port "$(E2E_SLOT_PORT_RANGE):$(E2E_SLOT_PORT_RANGE)@loadbalancer" \
	        --k3s-arg "--disable=traefik@server:*" \
	        --wait; \
	fi
	$(tools/kubectl) cluster-info --context k3d-$(E2E_CLUSTER)

.PHONY: e2e/cluster-down
e2e/cluster-down: $(tools/k3d)
	$(tools/k3d) cluster delete $(E2E_CLUSTER) || true

.PHONY: e2e/load
e2e/load: $(tools/k3d)
	@if ! test -s $(E2E_VERSION_FILE); then \
	    echo "error: $(E2E_VERSION_FILE) not found. Run 'make images' first." >&2; \
	    exit 1; \
	fi
	@tag="$$(head -n1 $(E2E_VERSION_FILE))-$(ARCH)"; \
	for c in $(E2E_COMPONENTS); do \
	    img="ghcr.io/emissary-ingress/$$c:$$tag"; \
	    echo "+ k3d image import $$img"; \
	    $(tools/k3d) image import "$$img" -c $(E2E_CLUSTER) || exit 1; \
	done

.PHONY: e2e/install
e2e/install: $(CRDS_CHART) $(EMISSARY_CHART) e2e/load $(tools/kubectl)
	helm upgrade --install emissary-crds $(CRDS_CHART) \
	    --namespace $(E2E_CRD_NAMESPACE) --create-namespace \
	    --wait --timeout 3m
	@tag="$$(head -n1 $(E2E_VERSION_FILE))-$(ARCH)"; \
	helm upgrade --install emissary-ingress $(EMISSARY_CHART) \
	    --namespace $(E2E_NAMESPACE) --create-namespace \
	    -f $(OSS_HOME)/test/e2e/helm-values.yaml \
	    --set image.repository=ghcr.io/emissary-ingress/emissary \
	    --set image.tag="$$tag" \
	    --set image.pullPolicy=IfNotPresent \
	    --set replicaCount=1 \
	    --set createDefaultListeners=true \
	    --wait --timeout 3m
	$(tools/kubectl) -n $(E2E_NAMESPACE) rollout status deploy/emissary-ingress --timeout=2m
	$(MAKE) e2e/install-slots

# Slot installs live in slots.sh so CI shares the same definition.
.PHONY: e2e/install-slots
e2e/install-slots:
	@if ! test -s $(E2E_VERSION_FILE); then \
	    echo "error: $(E2E_VERSION_FILE) not found. Run 'make images' first." >&2; \
	    exit 1; \
	fi
	@tag="$$(head -n1 $(E2E_VERSION_FILE))-$(ARCH)"; \
	E2E_NAMESPACE=$(E2E_NAMESPACE) \
	    $(E2E_SLOTS_SH) install $(EMISSARY_CHART) $(E2E_IMAGE_REPO) "$$tag"

.PHONY: e2e/uninstall-slots
e2e/uninstall-slots:
	@E2E_NAMESPACE=$(E2E_NAMESPACE) $(E2E_SLOTS_SH) uninstall

tools/kat-client = $(OSS_HOME)/tools/bin/kat-client
.PHONY: $(tools/kat-client)
$(tools/kat-client):
	@mkdir -p $(@D)
	cd $(OSS_HOME) && go build -o $@ ./cmd/kat-client

# Fan out one chainsaw process per slot plus one for `default`; see
# slots.sh for why each slot's shard runs serially.
.PHONY: e2e/run
e2e/run: $(tools/kat-client) $(tools/chainsaw)
	@if ! test -s $(E2E_VERSION_FILE); then \
	    echo "error: $(E2E_VERSION_FILE) not found. Run 'make images' first." >&2; \
	    exit 1; \
	fi
	@tag="$$(head -n1 $(E2E_VERSION_FILE))-$(ARCH)"; \
	KAT_CLIENT=$(tools/kat-client) \
	KAT_SERVER_IMAGE="ghcr.io/emissary-ingress/kat-server:$$tag" \
	E2E_NAMESPACE=$(E2E_NAMESPACE) \
	E2E_GATEWAY_URL=$(E2E_GATEWAY_URL) \
	    $(E2E_SLOTS_SH) run \
	        $(tools/chainsaw) \
	        $(OSS_HOME)/test/e2e/.chainsaw.yaml \
	        $(OSS_HOME)/test/e2e/fixtures

# Run a single fixture, e.g. `make e2e/run/http-basic`. A `shared` fixture goes
# to the base Emissary; an `exclusive` one is given the first slot, since with
# only one fixture running there is nothing to deal around. Override which slot
# with E2E_SLOT=slot5 if that one is busy.
.PHONY: e2e/run/%
e2e/run/%: $(tools/kat-client) $(tools/chainsaw)
	@if ! test -s $(E2E_VERSION_FILE); then \
	    echo "error: $(E2E_VERSION_FILE) not found. Run 'make images' first." >&2; \
	    exit 1; \
	fi
	@if ! test -f $(OSS_HOME)/test/e2e/fixtures/$*/chainsaw-test.yaml; then \
	    echo "error: no fixture named '$*' under test/e2e/fixtures/" >&2; \
	    exit 1; \
	fi
	@tag="$$(head -n1 $(E2E_VERSION_FILE))-$(ARCH)"; \
	kind="$$(sed -n 's/^ *slot: *//p' $(OSS_HOME)/test/e2e/fixtures/$*/chainsaw-test.yaml | head -n1)"; \
	url="$(E2E_GATEWAY_URL)"; tcp_url="$(E2E_GATEWAY_URL):6789"; slot=default; \
	if test "$$kind" = "exclusive"; then \
	    want="$${E2E_SLOT:-}"; \
	    while read -r name base; do \
	        if test -z "$$want" -o "$$name" = "$$want"; then \
	            slot="$$name"; \
	            url="$(E2E_GATEWAY_URL):$$base"; tcp_url="$(E2E_GATEWAY_URL):$$((base+2))"; \
	            break; \
	        fi; \
	    done <<<"$$($(E2E_SLOTS_SH) list)"; \
	fi; \
	echo "+ fixture '$*' ($${kind:-?}) on slot '$$slot' ($$url)"; \
	SLOT="$$slot" \
	KAT_CLIENT=$(tools/kat-client) \
	KAT_SERVER_IMAGE="ghcr.io/emissary-ingress/kat-server:$$tag" \
	GATEWAY_URL="$$url" \
	GATEWAY_TCP_URL="$$tcp_url" \
	    $(tools/chainsaw) test \
	        --config $(OSS_HOME)/test/e2e/.chainsaw.yaml \
	        $(OSS_HOME)/test/e2e/fixtures/$*

# Short VERSION override used for the local install path. Dirty trees produce
# long version strings that exceed chart-label length limits in Kubernetes.
# Override by setting E2E_LOCAL_VERSION on the command line if needed.
E2E_LOCAL_VERSION ?= v4.0.0-local

.PHONY: e2e/local
e2e/local:
	$(MAKE) e2e/cluster-up
	$(MAKE) images
	$(MAKE) VERSION=$(E2E_LOCAL_VERSION) e2e/install
	$(MAKE) e2e/run
