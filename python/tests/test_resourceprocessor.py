import sys

import pytest

from ambassador import Config
from ambassador.config.resourceprocessor import (
    ResourceKind,
    ResourceDict,
    ResourceEmission,
    LocationManager,
    ResourceProcessor,
    AggregateResourceProcessor,
    CounterResourceProcessor,
    DeduplicatingResourceProcessor,
)
from ambassador.utils import parse_yaml


def resource_dict_from_yaml(yaml: str, **kwargs) -> ResourceDict:
    return ResourceDict(parse_yaml(yaml)[0], **kwargs)


valid_knative_ingress = resource_dict_from_yaml('''
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

valid_mapping = resource_dict_from_yaml('''
---
apiVersion: getambassador.io/v2
kind: Mapping
metadata:
  name: test
spec:
    prefix: /test/
    service: test.default
''', default_namespace='default')


class TestResourceKind:

    def test_legacy(self):
        rk = ResourceKind('v1', 'Service')

        assert rk.api_version == 'v1'
        assert rk.kind == 'Service'
        assert rk.api_group is None
        assert rk.version == 'v1'
        assert rk.domain == 'service'

    def test_group(self):
        rk = ResourceKind.for_ambassador('Mapping')

        assert rk.api_version == 'getambassador.io/v2'
        assert rk.kind == 'Mapping'
        assert rk.api_group == 'getambassador.io'
        assert rk.version == 'v2'
        assert rk.domain == 'mapping.getambassador.io'


class TestResourceDict:

    def test_valid(self):
        assert valid_knative_ingress.kind == ResourceKind.for_knative_networking('Ingress')
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
        with pytest.raises(ValueError, match='not a valid Kubernetes resource'):
            resource_dict_from_yaml('apiVersion: v1')


class TestResourceEmission:

    def test_from_resource(self):
        emission = ResourceEmission.from_resource(valid_mapping)

        assert emission.rkey == f'{valid_mapping.name}.{valid_mapping.namespace}'
        assert emission.object['apiVersion'] == valid_mapping.kind.api_version
        assert emission.object['kind'] == valid_mapping.kind.kind
        assert emission.object['name'] == valid_mapping.name
        assert emission.object['namespace'] == valid_mapping.namespace
        assert emission.object['generation'] == valid_mapping.generation
        assert len(emission.object['labels']) == 1
        assert emission.object['labels']['ambassador_crd'] == emission.rkey
        assert emission.object['prefix'] == valid_mapping.spec['prefix']
        assert emission.object['service'] == valid_mapping.spec['service']


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


class FinalizingResourceProcessor (ResourceProcessor):

    finalize: bool = False

    def finalize(self):
        self.finalized = True


class TestAggregateResourceProcessor:

    def test_aggregation(self):
        aconf = Config()

        frp = FinalizingResourceProcessor()

        rp = AggregateResourceProcessor([
            CounterResourceProcessor(aconf, valid_knative_ingress.kind, 'test_1'),
            CounterResourceProcessor(aconf, valid_mapping.kind, 'test_2'),
            frp,
        ])

        assert len(rp.kinds()) == 2

        assert rp.try_process(valid_knative_ingress)
        assert rp.try_process(valid_mapping)

        assert aconf.get_count('test_1') == 1
        assert aconf.get_count('test_2') == 1

        rp.finalize()
        assert frp.finalized, 'Aggregate resource processor did not call finalizers'


class TestDeduplicatingResourceProcessor:

    def test_deduplication(self):
        aconf = Config()

        rp = DeduplicatingResourceProcessor(CounterResourceProcessor(aconf, valid_mapping.kind, 'test'))

        assert rp.try_process(valid_mapping)
        assert rp.try_process(valid_mapping)
        assert rp.try_process(valid_mapping)

        assert aconf.get_count('test') == 1


class TestCounterResourceProcessor:

    def test_count(self):
        aconf = Config()

        rp = CounterResourceProcessor(aconf, valid_mapping.kind, 'test')

        assert rp.try_process(valid_mapping), 'Resource processor rejected matching resource'
        assert rp.try_process(valid_mapping), 'Resource processor rejected matching resource (again)'
        assert not rp.try_process(valid_knative_ingress), 'Resource processor accepted non-matching resource'

        assert aconf.get_count('test') == 2, 'Resource processor did not increment counter'


if __name__ == '__main__':
    pytest.main(sys.argv)
