import sys

import logging
import pytest

from ambassador import Config
from ambassador.config.resourceprocessor import (
    LocationManager,
    ResourceManager,
    KubernetesProcessor,
    KubernetesGVK,
    KubernetesObject,
    AggregateKubernetesProcessor,
    CountingKubernetesProcessor,
    DeduplicatingKubernetesProcessor,
    AmbassadorProcessor,
)
from ambassador.utils import parse_yaml

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s test %(levelname)s: %(message)s",
    datefmt='%Y-%m-%d %H:%M:%S'
)

logger = logging.getLogger("ambassador")


def k8s_object_from_yaml(yaml: str, **kwargs) -> KubernetesObject:
    return KubernetesObject(parse_yaml(yaml)[0], **kwargs)


valid_knative_ingress = k8s_object_from_yaml('''
---
apiVersion: networking.internal.knative.dev/v1alpha1
kind: Ingress
metadata:
  annotations:
    getambassador.io/ambassador-id: webhook
    networking.knative.dev/ingress.class: ambassador.ingress.networking.knative.dev
  generation: 2
  labels:
    serving.knative.dev/route: helloworld-go
    serving.knative.dev/routeNamespace: test
    serving.knative.dev/service: helloworld-go
  name: helloworld-go
  namespace: test
spec:
  rules:
  - hosts:
    - helloworld-go.test.svc.cluster.local
    http:
      paths:
      - retries:
          attempts: 3
          perTryTimeout: 10m0s
        splits:
        - appendHeaders:
            Knative-Serving-Namespace: test
            Knative-Serving-Revision: helloworld-go-qf94m
          percent: 100
          serviceName: helloworld-go-qf94m
          serviceNamespace: test
          servicePort: 80
        timeout: 10m0s
    visibility: ClusterLocal
  visibility: ExternalIP
status:
  loadBalancer:
    ingress:
    - domainInternal: ambassador.ambassador-webhook.svc.cluster.local
  observedGeneration: 2
''')

valid_mapping = k8s_object_from_yaml('''
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: test
spec:
  prefix: /test/
  service: test.default
''', default_namespace='default')

valid_mapping_v1 = k8s_object_from_yaml('''
---
apiVersion: getambassador.io/v1
kind: Mapping
metadata:
  name: test
spec:
  prefix: /test/
  service: test.default
''', default_namespace='default')


class TestKubernetesGVK:

    def test_legacy(self):
        gvk = KubernetesGVK('v1', 'Service')

        assert gvk.api_version == 'v1'
        assert gvk.kind == 'Service'
        assert gvk.api_group is None
        assert gvk.version == 'v1'
        assert gvk.domain == 'service'

    def test_group(self):
        gvk = KubernetesGVK.for_ambassador('Mapping')

        assert gvk.api_version == 'getambassador.io/v2'
        assert gvk.kind == 'Mapping'
        assert gvk.api_group == 'getambassador.io'
        assert gvk.version == 'v2'
        assert gvk.domain == 'mapping.getambassador.io'


class TestKubernetesObject:

    def test_valid(self):
        assert valid_knative_ingress.gvk == KubernetesGVK.for_knative_networking('Ingress')
        assert valid_knative_ingress.namespace == 'test'
        assert valid_knative_ingress.name == 'helloworld-go'
        assert valid_knative_ingress.generation == 2
        assert len(valid_knative_ingress.annotations) == 2
        assert valid_knative_ingress.ambassador_id == 'webhook'
        assert len(valid_knative_ingress.labels) == 3
        assert valid_knative_ingress.spec['rules'][0]['hosts'][0] == 'helloworld-go.test.svc.cluster.local'
        assert valid_knative_ingress.status['observedGeneration'] == 2

        # Test default namespace fallback:
        assert valid_mapping.namespace == 'default'

    def test_invalid(self):
        with pytest.raises(ValueError, match='not a valid Kubernetes object'):
            k8s_object_from_yaml('apiVersion: v1')


class TestNormalizedResource:

    def test_kubernetes_object_conversion(self):
        resource = valid_mapping.as_normalized_resource()

        assert resource.rkey == f'{valid_mapping.name}.{valid_mapping.namespace}'
        assert resource.object['apiVersion'] == valid_mapping.gvk.api_version
        assert resource.object['kind'] == valid_mapping.gvk.kind
        assert resource.object['name'] == valid_mapping.name
        assert resource.object['namespace'] == valid_mapping.namespace
        assert resource.object['generation'] == valid_mapping.generation
        assert len(resource.object['labels']) == 1
        assert resource.object['labels']['ambassador_crd'] == resource.rkey
        assert resource.object['prefix'] == valid_mapping.spec['prefix']
        assert resource.object['service'] == valid_mapping.spec['service']


class TestLocationManager:

    def test_context_manager(self):
        lm = LocationManager()

        assert len(lm.previous) == 0

        assert lm.current.filename is None
        assert lm.current.ocount == 1

        with lm.push(filename='test', ocount=2) as loc:
            assert len(lm.previous) == 1
            assert lm.current == loc

            assert loc.filename == 'test'
            assert loc.ocount == 2

            with lm.push_reset() as rloc:
                assert len(lm.previous) == 2
                assert lm.current == rloc

                assert rloc.filename == 'test'
                assert rloc.ocount == 1

        assert len(lm.previous) == 0

        assert lm.current.filename is None
        assert lm.current.ocount == 1


class FinalizingKubernetesProcessor (KubernetesProcessor):

    finalized: bool = False

    def finalize(self):
        self.finalized = True


class TestAmbassadorProcessor:

    def test_mapping(self):
        aconf = Config()
        mgr = ResourceManager(logger, aconf)

        assert AmbassadorProcessor(mgr).try_process(valid_mapping)
        assert len(mgr.elements) == 1

        aconf.load_all(mgr.elements)
        assert len(aconf.errors) == 0

        mappings = aconf.get_config('mappings')
        assert len(mappings) == 1

        mapping = next(iter(mappings.values()))
        assert mapping.apiVersion == valid_mapping.gvk.api_version
        assert mapping.name == valid_mapping.name
        assert mapping.namespace == valid_mapping.namespace
        assert mapping.prefix == valid_mapping.spec['prefix']
        assert mapping.service == valid_mapping.spec['service']

    def test_mapping_v1(self):
        aconf = Config()
        mgr = ResourceManager(logger, aconf)

        assert AmbassadorProcessor(mgr).try_process(valid_mapping_v1)
        assert len(mgr.elements) == 1

        aconf.load_all(mgr.elements)
        assert len(aconf.errors) == 0

        mappings = aconf.get_config('mappings')
        assert len(mappings) == 1

        mapping = next(iter(mappings.values()))
        assert mapping.apiVersion == valid_mapping_v1.gvk.api_version
        assert mapping.name == valid_mapping_v1.name
        assert mapping.namespace == valid_mapping_v1.namespace
        assert mapping.prefix == valid_mapping_v1.spec['prefix']
        assert mapping.service == valid_mapping_v1.spec['service']


class TestAggregateKubernetesProcessor:

    def test_aggregation(self):
        aconf = Config()

        fp = FinalizingKubernetesProcessor()

        p = AggregateKubernetesProcessor([
            CountingKubernetesProcessor(aconf, valid_knative_ingress.gvk, 'test_1'),
            CountingKubernetesProcessor(aconf, valid_mapping.gvk, 'test_2'),
            fp,
        ])

        assert len(p.kinds()) == 2

        assert p.try_process(valid_knative_ingress)
        assert p.try_process(valid_mapping)

        assert aconf.get_count('test_1') == 1
        assert aconf.get_count('test_2') == 1

        p.finalize()
        assert fp.finalized, 'Aggregate processor did not call finalizers'


class TestDeduplicatingKubernetesProcessor:

    def test_deduplication(self):
        aconf = Config()

        p = DeduplicatingKubernetesProcessor(CountingKubernetesProcessor(aconf, valid_mapping.gvk, 'test'))

        assert p.try_process(valid_mapping)
        assert p.try_process(valid_mapping)
        assert p.try_process(valid_mapping)

        assert aconf.get_count('test') == 1


class TestCountingKubernetesProcessor:

    def test_count(self):
        aconf = Config()

        p = CountingKubernetesProcessor(aconf, valid_mapping.gvk, 'test')

        assert p.try_process(valid_mapping), 'Processor rejected matching resource'
        assert p.try_process(valid_mapping), 'Processor rejected matching resource (again)'
        assert not p.try_process(valid_knative_ingress), 'Processor accepted non-matching resource'

        assert aconf.get_count('test') == 2, 'Processor did not increment counter'


if __name__ == '__main__':
    pytest.main(sys.argv)
