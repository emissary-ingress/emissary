include build-aux/tools.mk

#
# Auxiliary Docker images needed for the tests

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
	@echo "" >> "$@"	# Make sure the ID file ends with a newline.
clean: $(foreach svc,$(test_svcs),docker/test-$(svc).docker.clean)

# kat-client.docker
docker/kat-client.go.layer.tar: $(tools/ocibuild) $(tools/write-ifchanged) FORCE
	@echo "==== docker/kat-client.go.layer.tar in check.mk, as $@: $^"
	GOFLAGS=-mod=mod $(tools/ocibuild) layer gobuild ./cmd/kat-client | $(tools/write-ifchanged) $@

docker/kat-client.fs.layer.tar: $(tools/ocibuild) $(tools/write-ifchanged) FORCE
	@echo "==== docker/kat-client.fs.layer.tar in check.mk, as $@: $^"
	{ $(tools/ocibuild) layer dir \
	  --prefix=work \
	  --chown-uid=0 --chown-uname=root \
	  --chown-gid=0 --chown-uname=root \
	  docker/kat-client; } | $(tools/write-ifchanged) $@

docker/.kat-client.img.tar.stamp: $(tools/ocibuild) docker/base.img.tar docker/kat-client.go.layer.tar docker/kat-client.fs.layer.tar
	@echo "==== docker/.kat-client.img.tar.stamp in check.mk, as $@: $^"
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
docker/kat-server.img.tar.clean: docker/kat-server.rm-r

#
# Helm tests

test-chart-values.yaml: docker/emissary.docker.push.remote build-aux/check.mk
	{ \
	  echo 'test:'; \
	  echo '  enabled: true'; \
	  echo 'image:'; \
	  sed -E -n '2s/^(.*):.*/  repository: \1/p' < $<; \
	  sed -E -n '2s/.*:/  tag: /p' < $<; \
	} >$@
clean: test-chart-values.yaml.rm
build-output/chart-%/ci: build-output/chart-% test-chart-values.yaml
	rm -rf $@
	cp -a $@.in $@
	for file in $@/*-values.yaml; do cat test-chart-values.yaml >> "$$file"; done

test-chart: $(tools/ct) $(tools/kubectl) $(chart_dir)/ci build-output/yaml-$(patsubst v%,%,$(VERSION)) $(if $(DEV_USE_IMAGEPULLSECRET),push-pytest-images $(OSS_HOME)/venv)
ifneq ($(DEV_USE_IMAGEPULLSECRET),)
	. venv/bin/activate && KUBECONFIG=$(DEV_KUBECONFIG) python3 -c 'from tests.integration.utils import install_crds; install_crds()'
else
	$(tools/kubectl) --kubeconfig=$(DEV_KUBECONFIG) apply -f build-output/yaml-$(patsubst v%,%,$(VERSION))/emissary-crds.yaml
endif
	$(tools/kubectl) --kubeconfig=$(DEV_KUBECONFIG) --namespace=emissary-system wait --timeout=90s --for=condition=available Deployments/emissary-apiext
	cd $(chart_dir) && KUBECONFIG=$(DEV_KUBECONFIG) $(abspath $(tools/ct)) install --config=./ct.yaml
.PHONY: test-chart

#
# Other

clean: .pytest_cache.rm-r .coverage.rm

dtest.clean:
	docker container list --filter=label=scope=dtest --quiet | xargs -r docker container kill
clean: dtest.clean
