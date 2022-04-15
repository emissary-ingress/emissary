include build-aux/tools.mk

#
# Auxiliar Docker images needed for the tests

# Keep this list in-sync with python/tests/integration/manifests.py
push-pytest-images: docker/emissary.docker.push.remote
push-pytest-images: docker/test-auth.docker.push.remote
push-pytest-images: docker/test-shadow.docker.push.remote
push-pytest-images: docker/test-stats.docker.push.remote
push-pytest-images: docker/kat-client.docker.push.remote
push-pytest-images: docker/kat-server.docker.push.remote
.PHONY: push-pytest-images

# test-{auth,shadow,stats}.docker
test_svcs = auth shadow stats
$(foreach svc,$(test_svcs),docker/.test-$(svc).docker.stamp): docker/.%.docker.stamp: docker/%/Dockerfile FORCE
	docker build --iidfile=$@ $(<D)

# kat-client.docker
docker/kat-client.go.layer.tar: $(tools/ocibuild) $(tools/write-ifchanged) FORCE
	GOFLAGS=-mod=mod $(tools/ocibuild) layer gobuild ./cmd/kat-client | $(tools/write-ifchanged) $@
docker/kat-client.fs.layer.tar: $(tools/ocibuild) $(tools/write-ifchanged) FORCE
	{ $(tools/ocibuild) layer dir \
	  --prefix=work \
	  --chown-uid=0 --chown-uname=root \
	  --chown-gid=0 --chown-uname=root \
	  docker/kat-client; } | $(tools/write-ifchanged) $@
docker/.kat-client.img.tar.stamp: $(tools/ocibuild) docker/base.img.tar docker/kat-client.go.layer.tar docker/kat-client.fs.layer.tar
	{ $(tools/ocibuild) image build \
	  --base=docker/base.img.tar \
	  --config.Cmd='sleep' --config.Cmd='3600' \
	  --tag=emissary.local/kat-client:latest \
	  <($(tools/ocibuild) layer squash $(filter %.layer.tar,$^)); } > $@

# kat-server.docker
docker/kat-server.go.layer.tar: $(tools/ocibuild) $(tools/write-ifchanged) FORCE
	GOFLAGS=-mod=mod $(tools/ocibuild) layer gobuild ./cmd/kat-server | $(tools/write-ifchanged) $@
docker/kat-server.certs.layer.tar: $(tools/ocibuild) $(tools/write-ifchanged) docker/kat-server/server.crt docker/kat-server/server.key
	$(tools/ocibuild) layer dir --prefix=work docker/kat-server | $(tools/write-ifchanged) $@
docker/kat-server/server.crt: $(tools/testcert-gen)
	mkdir -p $(@D)
	$(tools/testcert-gen) --out-cert=$@ --out-key=/dev/null --hosts=kat-server.test.getambassador.io
docker/kat-server/server.key: $(tools/testcert-gen)
	mkdir -p $(@D)
	$(tools/testcert-gen) --out-cert=/dev/null --out-key=$@ --hosts=kat-server.test.getambassador.io
docker/.kat-server.img.tar.stamp: $(tools/ocibuild) docker/base.img.tar docker/kat-server.go.layer.tar docker/kat-server.certs.layer.tar
	{ $(tools/ocibuild) image build \
	  --base=docker/base.img.tar \
	  --config.Env.append=GRPC_VERBOSITY=debug \
	  --config.Env.append=GRPC_TRACE=tcp,http,api \
	  --config.WorkingDir='/work' \
	  --config.Cmd='kat-server' \
	  --tag=emissary.local/kat-server:latest \
	  <($(tools/ocibuild) layer squash $(filter %.layer.tar,$^)); } > $@

#
# Helm tests

test-chart-values.yaml: docker/emissary.docker.push.remote
	{ \
	  echo 'image:'; \
	  sed -E -n '2s/^(.*):.*/  repository: \1/p' < $<; \
	  sed -E -n '2s/.*:/  tag: /p' < $<; \
	} >$@
charts/emissary-ingress/ci: %: %.in test-chart-values.yaml
	rm -rf $@
	cp -a $@.in $@
	for file in $@/*-values.yaml; do cat test-chart-values.yaml >> "$$file"; done

test-chart: $(tools/ct) $(tools/kubectl) charts/emissary-ingress/ci $(if $(DEV_USE_IMAGEPULLSECRET),push-pytest-images $(OSS_HOME)/venv)
ifneq ($(DEV_USE_IMAGEPULLSECRET),)
	. venv/bin/activate && KUBECONFIG=$(DEV_KUBECONFIG) python3 -c 'from tests.integration.utils import install_crds; install_crds()'
else
	$(tools/kubectl) --kubeconfig=$(DEV_KUBECONFIG) apply -f manifests/emissary/emissary-crds.yaml
endif
	$(tools/kubectl) --kubeconfig=$(DEV_KUBECONFIG) --namespace=emissary-system wait --timeout=90s --for=condition=available Deployments/emissary-apiext
	cd charts/emissary-ingress && KUBECONFIG=$(DEV_KUBECONFIG) $(abspath $(tools/ct)) install --config=./ct.yaml
.PHONY: test-chart
