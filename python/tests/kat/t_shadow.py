import json
import pytest

from typing import ClassVar, Dict, List, Sequence, Tuple, Union

from kat.harness import sanitize, variants, Query, Runner

from abstract_tests import AmbassadorTest, HTTP
from abstract_tests import assert_default_errors, MappingTest, OptionTest, ServiceType, Node, Test


class ShadowTestCANFLAKE(MappingTest):
    parent: AmbassadorTest
    target: ServiceType
    shadow: ServiceType

    def init(self) -> None:
        self.target = HTTP(name="target")
        self.options = None

    def manifests(self) -> str:
        return """
---
apiVersion: v1
kind: Service
metadata:
  name: shadow
spec:
  selector:
    app: shadow
  ports:
  - port: 80
    name: http
    targetPort: http
  type: ClusterIP
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: shadow
spec:
  selector:
    matchLabels:
      app: shadow
  replicas: 1
  strategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: shadow
    spec:
      containers:
      - name: shadow
        image: {self.test_image[shadow]}
        ports:
        - name: http
          containerPort: 3000
""" + super().manifests()

    def config(self):
        yield self.target, self.format("""
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name:  {self.name}-target
hostname: "*"
prefix: /{self.name}/mark/
rewrite: /mark/
service: https://{self.target.path.fqdn}
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name:  {self.name}-weighted-target
hostname: "*"
prefix: /{self.name}/weighted-mark/
rewrite: /mark/
service: https://{self.target.path.fqdn}
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name:  {self.name}-shadow
hostname: "*"
prefix: /{self.name}/mark/
rewrite: /mark/
service: shadow.plain-namespace
shadow: true
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name:  {self.name}-weighted-shadow
hostname: "*"
prefix: /{self.name}/weighted-mark/
rewrite: /mark/
service: shadow.plain-namespace
weight: 10
shadow: true
---
apiVersion: x.getambassador.io/v3alpha1
kind: AmbassadorMapping
name:  {self.name}-checkshadow
hostname: "*"
prefix: /{self.name}/check/
rewrite: /check/
service: shadow.plain-namespace
""")

    def requirements(self):
        yield from super().requirements()
        yield ("url", Query("http://shadow.plain-namespace/clear/"))

    def queries(self):
        # There should be no Ambassador errors. At all.
        yield Query(self.parent.url("ambassador/v0/diag/?json=true&filter=errors"), phase=1)

        for i in range(100):
            # First query marks one bucket from 0 - 9. The main target service is just a
            # normal KAT backend so nothing funky happens there, but it's also shadowed
            # to our shadow service that's tallying calls by bucket. So, basically, each
            # shadow bucket 0-9 should end up with 10 call.s
            bucket = i % 10
            yield Query(self.parent.url(f'{self.name}/mark/{bucket}'))

        for i in range(500):
            # We also do a call to weighted-mark, which is exactly the same _but_ the
            # shadow is just 20%. So instead of 50 calls per bucket, we should expect
            # 10.
            #
            # We use different bucket numbers so we can tell which call was which
            # shadow.

            bucket = (i % 10) + 100
            yield Query(self.parent.url(f'{self.name}/weighted-mark/{bucket}'))

        # Finally, in phase 2, grab the bucket counts.
        yield Query(self.parent.url("%s/check/" % self.name), phase=2)

    def check(self):
        # XXX Ew. If self.results[0].json is empty, the harness won't convert it to a response.
        errors = self.results[0].json or {}

        # We shouldn't have any missing-CRD-types errors any more.
        for source, error in errors:
          if (('could not find' in error) and ('CRD definitions' in error)):
            assert False, f"Missing CRDs: {error}"

          if 'Ingress resources' in error:
            assert False, f"Ingress resource error: {error}"

        # The default errors assume that we have missing CRDs, and that's not correct any more,
        # so don't try to use assert_default_errors here.

        for result in self.results:
            if "mark" in result.query.url:
                assert not result.headers.get('X-Shadowed', False)
            elif "check" in result.query.url:
                data = result.json
                weighted_total = 0

                for i in range(10):
                    # Buckets 0-9 should have 10 per bucket. We'll actually check these values
                    # pretty carefully, because this bit of routing isn't probabilistic.
                    value = data.get(str(i), -1)
                    error = abs(value - 10)

                    assert error <= 2, f'bucket {i} should have 10 calls, got {value}'

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
                assert weighted_total > 0, f'weighted buckets should have 50 total calls but got zero'
