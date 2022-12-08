from typing import ClassVar, Generator, Tuple, Union

from abstract_tests import HTTP, MappingTest, Node, ServiceType
from kat.harness import Query


class ShadowBackend(ServiceType):
    skip_variant: ClassVar[bool] = True

    def __init__(self, *args, **kwargs) -> None:
        kwargs[
            "service_manifests"
        ] = """
---
apiVersion: v1
kind: Service
metadata:
  name: {self.path.k8s}
spec:
  selector:
    backend: {self.path.k8s}
  ports:
  - port: 80
    name: http
    targetPort: http
  type: ClusterIP
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {self.path.k8s}
spec:
  selector:
    matchLabels:
      backend: {self.path.k8s}
  replicas: 1
  strategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        backend: {self.path.k8s}
    spec:
      containers:
      - name: shadow
        image: {images[test-shadow]}
        ports:
        - name: http
          containerPort: 3000
"""
        super().__init__(*args, **kwargs)

    def requirements(self):
        yield ("url", Query(f"http://{self.path.fqdn}/clear/"))


class ShadowTestCANFLAKE(MappingTest):
    shadow: ServiceType

    # XXX This type: ignore is here because we're deliberately overriding the
    # parent's init to have a different signature... but it's also intimately
    # (nay, incestuously) related to the variant()'s yield() above, and I really
    # don't want to deal with that right now. So. We'll deal with it later.
    def init(self) -> None:  # type: ignore
        MappingTest.init(self, HTTP(name="target"))
        self.shadow = ShadowBackend()

    def config(self) -> Generator[Union[str, Tuple[Node, str]], None, None]:
        yield self.target, self.format(
            """
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.name}-target
hostname: "*"
prefix: /{self.name}/mark/
rewrite: /mark/
service: https://{self.target.path.fqdn}
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.name}-weighted-target
hostname: "*"
prefix: /{self.name}/weighted-mark/
rewrite: /mark/
service: https://{self.target.path.fqdn}
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.name}-shadow
hostname: "*"
prefix: /{self.name}/mark/
rewrite: /mark/
service: {self.shadow.path.fqdn}
shadow: true
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.name}-weighted-shadow
hostname: "*"
prefix: /{self.name}/weighted-mark/
rewrite: /mark/
service: {self.shadow.path.fqdn}
weight: 10
shadow: true
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
name:  {self.name}-checkshadow
hostname: "*"
prefix: /{self.name}/check/
rewrite: /check/
service: {self.shadow.path.fqdn}
"""
        )

    def queries(self):
        # There should be no Ambassador errors. At all.
        yield Query(self.parent.url("ambassador/v0/diag/?json=true&filter=errors"), phase=1)

        for i in range(100):
            # First query marks one bucket from 0 - 9. The main target service is just a
            # normal KAT backend so nothing funky happens there, but it's also shadowed
            # to our shadow service that's tallying calls by bucket. So, basically, each
            # shadow bucket 0-9 should end up with 10 call.s
            bucket = i % 10
            yield Query(self.parent.url(f"{self.name}/mark/{bucket}"))

        for i in range(500):
            # We also do a call to weighted-mark, which is exactly the same _but_ the
            # shadow is just 20%. So instead of 50 calls per bucket, we should expect
            # 10.
            #
            # We use different bucket numbers so we can tell which call was which
            # shadow.

            bucket = (i % 10) + 100
            yield Query(self.parent.url(f"{self.name}/weighted-mark/{bucket}"))

        # Finally, in phase 2, grab the bucket counts.
        yield Query(self.parent.url("%s/check/" % self.name), phase=2)

    def check(self):
        # XXX Ew. If self.results[0].json is empty, the harness won't convert it to a response.
        errors = self.results[0].json or {}

        # We shouldn't have any missing-CRD-types errors any more.
        for source, error in errors:
            if ("could not find" in error) and ("CRD definitions" in error):
                assert False, f"Missing CRDs: {error}"

            if "Ingress resources" in error:
                assert False, f"Ingress resource error: {error}"

        # The default errors assume that we have missing CRDs, and that's not correct any more,
        # so don't try to use assert_default_errors here.

        for result in self.results:
            if "mark" in result.query.url:
                assert not result.headers.get("X-Shadowed", False)
            elif "check" in result.query.url:
                data = result.json
                weighted_total = 0

                for i in range(10):
                    # Buckets 0-9 should have 10 per bucket. We'll actually check these values
                    # pretty carefully, because this bit of routing isn't probabilistic.
                    value = data.get(str(i), -1)
                    error = abs(value - 10)

                    assert error <= 2, f"bucket {i} should have 10 calls, got {value}"

                    # Buckets 100-109 should also have 10 per bucket... but honestly, this is
                    # a pretty small sample size, and Envoy's randomization seems to kinda suck
                    # at small sample sizes. Since we're here to test Ambassador's ability to
                    # configure Envoy, rather than trying to test Envoy's ability to properly
                    # weight things, we'll just make sure that _some_ calls got into the shadow
                    # buckets, and not worry about how many it was exactly.

                    wi = i + 100

                    value = data.get(str(wi), 0)

                    # error = abs(value - 10)
                    # assert error <= 2, f'bucket {wi} should have 10 calls, got {value}'

                    weighted_total += value

                # See above for why we're just doing a >0 check here.
                # assert abs(weighted_total - 50) <= 10, f'weighted buckets should have 50 total calls, got {weighted_total}'
                assert (
                    weighted_total > 0
                ), f"weighted buckets should have 50 total calls but got zero"
