import logging
import os
import sys

import pytest

from ambassador.utils import NullSecretHandler, parse_bool

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s test %(levelname)s: %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)

logger = logging.getLogger("ambassador")

from ambassador import Config
from ambassador.fetch import ResourceFetcher
from ambassador.fetch.ambassador import AmbassadorProcessor
from ambassador.fetch.dependency import (
    DependencyManager,
    IngressClassesDependency,
    SecretDependency,
    ServiceDependency,
)
from ambassador.fetch.k8sobject import (
    KubernetesGVK,
    KubernetesObject,
    KubernetesObjectKey,
    KubernetesObjectScope,
)
from ambassador.fetch.k8sprocessor import (
    AggregateKubernetesProcessor,
    CountingKubernetesProcessor,
    DeduplicatingKubernetesProcessor,
    KubernetesProcessor,
)
from ambassador.fetch.location import LocationManager
from ambassador.fetch.resource import NormalizedResource, ResourceManager
from ambassador.utils import parse_yaml


def k8s_object_from_yaml(yaml: str) -> KubernetesObject:
    return KubernetesObject(parse_yaml(yaml)[0])


valid_knative_ingress = k8s_object_from_yaml(
    """
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
"""
)

valid_ingress_class = k8s_object_from_yaml(
    """
apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: external-lb
spec:
  controller: getambassador.io/ingress-controller
"""
)

valid_mapping = k8s_object_from_yaml(
    """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: test
  namespace: default
spec:
  hostname: "*"
  prefix: /test/
  service: test.default
"""
)

valid_mapping_v1 = k8s_object_from_yaml(
    """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: test
  namespace: default
spec:
  hostname: "*"
  prefix: /test/
  service: test.default
"""
)


class TestKubernetesGVK:
    def test_legacy(self):
        gvk = KubernetesGVK("v1", "Service")

        assert gvk.api_version == "v1"
        assert gvk.kind == "Service"
        assert gvk.api_group is None
        assert gvk.version == "v1"
        assert gvk.domain == "service"

    def test_group(self):
        gvk = KubernetesGVK.for_ambassador("Mapping", version="v3alpha1")

        assert gvk.api_version == "getambassador.io/v3alpha1"
        assert gvk.kind == "Mapping"
        assert gvk.api_group == "getambassador.io"
        assert gvk.version == "v3alpha1"
        assert gvk.domain == "mapping.getambassador.io"


class TestKubernetesObject:
    def test_valid(self):
        assert valid_knative_ingress.gvk == KubernetesGVK.for_knative_networking("Ingress")
        assert valid_knative_ingress.namespace == "test"
        assert valid_knative_ingress.name == "helloworld-go"
        assert valid_knative_ingress.scope == KubernetesObjectScope.NAMESPACE
        assert valid_knative_ingress.key == KubernetesObjectKey(
            valid_knative_ingress.gvk, "test", "helloworld-go"
        )
        assert valid_knative_ingress.generation == 2
        assert len(valid_knative_ingress.annotations) == 2
        assert valid_knative_ingress.ambassador_id == "webhook"
        assert len(valid_knative_ingress.labels) == 3
        assert (
            valid_knative_ingress.spec["rules"][0]["hosts"][0]
            == "helloworld-go.test.svc.cluster.local"
        )
        assert valid_knative_ingress.status["observedGeneration"] == 2

    def test_valid_cluster_scoped(self):
        assert valid_ingress_class.name == "external-lb"
        assert valid_ingress_class.scope == KubernetesObjectScope.CLUSTER
        assert valid_ingress_class.key == KubernetesObjectKey(
            valid_ingress_class.gvk, None, "external-lb"
        )
        assert valid_ingress_class.key.namespace is None

        with pytest.raises(AttributeError):
            valid_ingress_class.namespace

    def test_invalid(self):
        with pytest.raises(ValueError, match="not a valid Kubernetes object"):
            k8s_object_from_yaml("apiVersion: v1")


class TestNormalizedResource:
    def test_kubernetes_object_conversion(self):
        resource = NormalizedResource.from_kubernetes_object(valid_mapping)

        assert resource.rkey == f"{valid_mapping.name}.{valid_mapping.namespace}"
        assert resource.object["apiVersion"] == valid_mapping.gvk.api_version
        assert resource.object["kind"] == valid_mapping.kind
        assert resource.object["name"] == valid_mapping.name
        assert resource.object["namespace"] == valid_mapping.namespace
        assert resource.object["generation"] == valid_mapping.generation
        assert len(resource.object["metadata_labels"]) == 1
        assert resource.object["metadata_labels"]["ambassador_crd"] == resource.rkey
        assert resource.object["prefix"] == valid_mapping.spec["prefix"]
        assert resource.object["service"] == valid_mapping.spec["service"]


class TestLocationManager:
    def test_context_manager(self):
        lm = LocationManager()

        assert len(lm.previous) == 0

        assert lm.current.filename is None
        assert lm.current.ocount == 1

        with lm.push(filename="test", ocount=2) as loc:
            assert len(lm.previous) == 1
            assert lm.current == loc

            assert loc.filename == "test"
            assert loc.ocount == 2

            with lm.push_reset() as rloc:
                assert len(lm.previous) == 2
                assert lm.current == rloc

                assert rloc.filename == "test"
                assert rloc.ocount == 1

        assert len(lm.previous) == 0

        assert lm.current.filename is None
        assert lm.current.ocount == 1


class FinalizingKubernetesProcessor(KubernetesProcessor):

    finalized: bool = False

    def finalize(self):
        self.finalized = True


class TestAmbassadorProcessor:
    def test_mapping(self):
        aconf = Config()
        mgr = ResourceManager(logger, aconf, DependencyManager([]))

        assert AmbassadorProcessor(mgr).try_process(valid_mapping)
        assert len(mgr.elements) == 1

        aconf.load_all(mgr.elements)
        assert len(aconf.errors) == 0

        mappings = aconf.get_config("mappings")
        assert mappings
        assert len(mappings) == 1

        mapping = next(iter(mappings.values()))
        assert mapping.apiVersion == valid_mapping.gvk.api_version
        assert mapping.name == valid_mapping.name
        assert mapping.namespace == valid_mapping.namespace
        assert mapping.prefix == valid_mapping.spec["prefix"]
        assert mapping.service == valid_mapping.spec["service"]

    def test_mapping_v1(self):
        aconf = Config()
        mgr = ResourceManager(logger, aconf, DependencyManager([]))

        assert AmbassadorProcessor(mgr).try_process(valid_mapping_v1)
        assert len(mgr.elements) == 1
        print(f"mgr.elements[0]={mgr.elements[0].apiVersion}")

        aconf.load_all(mgr.elements)
        assert len(aconf.errors) == 0

        mappings = aconf.get_config("mappings")
        assert mappings
        assert len(mappings) == 1

        mapping = next(iter(mappings.values()))
        assert mapping.apiVersion == valid_mapping_v1.gvk.api_version
        assert mapping.name == valid_mapping_v1.name
        assert mapping.namespace == valid_mapping_v1.namespace
        assert mapping.prefix == valid_mapping_v1.spec["prefix"]
        assert mapping.service == valid_mapping_v1.spec["service"]

    def test_ingress_with_named_port(self):
        isEdgeStack = parse_bool(os.environ.get("EDGE_STACK", "false"))

        yaml = """
---
apiVersion: v1
kind: Service
metadata:
  name: quote
  namespace: default
spec:
  type: ClusterIP
  ports:
  - name: http
    port: 3000
    protocol: TCP
    targetPort: 3000
  selector:
    app: quote
---
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  annotations:
    getambassador.io/ambassador-id: default
    kubernetes.io/ingress.class: ambassador
  name: quote
  namespace: default
spec:
  rules:
  - http:
      paths:
      - path: /
        pathType: ImplementationSpecific
        backend:
          serviceName: quote
          servicePort: http
      - path: /metrics
        pathType: ImplementationSpecific
        backend:
          serviceName: quote
          servicePort: metrics
      - path: /health
        pathType: ImplementationSpecific
        backend:
          serviceName: quote
          servicePort: 9000
      - path: /missed-name
        pathType: ImplementationSpecific
        backend:
          serviceName: missed
          servicePort: missed
      - path: /missed-number
        pathType: ImplementationSpecific
        backend:
          serviceName: missed
          servicePort: 8080
status:
  loadBalancer: {}
"""
        aconf = Config()
        logger.setLevel(logging.DEBUG)

        fetcher = ResourceFetcher(logger, aconf)
        fetcher.parse_yaml(yaml, True)

        mgr = fetcher.manager

        expectedElements = 7 if isEdgeStack else 6
        assert len(mgr.elements) == expectedElements

        aconf.load_all(fetcher.sorted())
        assert len(aconf.errors) == 0

        mappings = aconf.get_config("mappings")
        assert mappings

        expectedMappings = 6 if isEdgeStack else 5
        assert len(mappings) == expectedMappings

        mapping_root = mappings.get("quote-0-0")
        assert mapping_root
        assert mapping_root.prefix == "/"
        assert mapping_root.service == "quote.default:3000"

        mapping_metrics = mappings.get("quote-0-1")
        assert mapping_metrics
        assert mapping_metrics.prefix == "/metrics"
        assert mapping_metrics.service == "quote.default:metrics"

        mapping_health = mappings.get("quote-0-2")
        assert mapping_health
        assert mapping_health.prefix == "/health"
        assert mapping_health.service == "quote.default:9000"

        mapping_missed_name = mappings.get("quote-0-3")
        assert mapping_missed_name
        assert mapping_missed_name.prefix == "/missed-name"
        assert mapping_missed_name.service == "missed.default:missed"

        mapping_missed_number = mappings.get("quote-0-4")
        assert mapping_missed_number
        assert mapping_missed_number.prefix == "/missed-number"
        assert mapping_missed_number.service == "missed.default:8080"


class TestAggregateKubernetesProcessor:
    def test_aggregation(self):
        aconf = Config()

        fp = FinalizingKubernetesProcessor()

        p = AggregateKubernetesProcessor(
            [
                CountingKubernetesProcessor(aconf, valid_knative_ingress.gvk, "test_1"),
                CountingKubernetesProcessor(aconf, valid_mapping.gvk, "test_2"),
                fp,
            ]
        )

        assert len(p.kinds()) == 2

        assert p.try_process(valid_knative_ingress)
        assert p.try_process(valid_mapping)

        assert aconf.get_count("test_1") == 1
        assert aconf.get_count("test_2") == 1

        p.finalize()
        assert fp.finalized, "Aggregate processor did not call finalizers"


class TestDeduplicatingKubernetesProcessor:
    def test_deduplication(self):
        aconf = Config()

        p = DeduplicatingKubernetesProcessor(
            CountingKubernetesProcessor(aconf, valid_mapping.gvk, "test")
        )

        assert p.try_process(valid_mapping)
        assert p.try_process(valid_mapping)
        assert p.try_process(valid_mapping)

        assert aconf.get_count("test") == 1


class TestCountingKubernetesProcessor:
    def test_count(self):
        aconf = Config()

        p = CountingKubernetesProcessor(aconf, valid_mapping.gvk, "test")

        assert p.try_process(valid_mapping), "Processor rejected matching resource"
        assert p.try_process(valid_mapping), "Processor rejected matching resource (again)"
        assert not p.try_process(valid_knative_ingress), "Processor accepted non-matching resource"

        assert aconf.get_count("test") == 2, "Processor did not increment counter"


class TestDependencyManager:
    def setup(self):
        self.deps = DependencyManager(
            [
                SecretDependency(),
                ServiceDependency(),
                IngressClassesDependency(),
            ]
        )

    def test_cyclic(self):
        a = self.deps.for_instance(object())
        b = self.deps.for_instance(object())
        c = self.deps.for_instance(object())

        a.provide(SecretDependency)
        a.want(ServiceDependency)
        b.provide(ServiceDependency)
        b.want(IngressClassesDependency)
        c.provide(IngressClassesDependency)
        c.want(SecretDependency)

        with pytest.raises(ValueError):
            self.deps.sorted_watt_keys()

    def test_sort(self):
        a = self.deps.for_instance(object())
        b = self.deps.for_instance(object())
        c = self.deps.for_instance(object())

        a.want(SecretDependency)
        a.want(ServiceDependency)
        a.provide(IngressClassesDependency)
        b.provide(SecretDependency)
        c.provide(ServiceDependency)

        assert self.deps.sorted_watt_keys() == ["secret", "service", "ingressclasses"]


if __name__ == "__main__":
    pytest.main(sys.argv)
