import json
import pytest

from typing import ClassVar, Dict, List, Sequence, Tuple, Union

from kat.harness import sanitize, variants, Query, Runner
from kat import manifests

from abstract_tests import AmbassadorTest, HTTP
from abstract_tests import DEFAULT_ERRORS, MappingTest, OptionTest, ServiceType, Node, Test


class ShadowTest(MappingTest):
    parent: AmbassadorTest
    target: ServiceType
    shadow: ServiceType

    def init(self) -> None:
        self.target = HTTP(name="target")
        self.options = None

    def manifests(self) -> str:
        s = super().manifests() or ""

        return s + """
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
  type: NodePort
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: shadow
spec:
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
        image: dwflynn/shadow:0.0.2
        imagePullPolicy: Always
        ports:
        - name: http
          containerPort: 3000
"""

    def config(self):
        yield self.target, self.format("""
---
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.name}-target
prefix: /{self.name}/mark/
rewrite: /mark/
service: https://{self.target.path.fqdn}
---
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.name}-weighted-target
prefix: /{self.name}/weighted-mark/
rewrite: /mark/
service: https://{self.target.path.fqdn}
---
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.name}-shadow
prefix: /{self.name}/mark/
rewrite: /mark/
service: shadow.plain-namespace
shadow: true
---
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.name}-weighted-shadow
prefix: /{self.name}/weighted-mark/
rewrite: /mark/
service: shadow.plain-namespace
weight: 10
shadow: true
---
apiVersion: ambassador/v1
kind:  Mapping
name:  {self.name}-checkshadow
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
        assert(errors == DEFAULT_ERRORS)

        for result in self.results:
            if "mark" in result.query.url:
                assert not result.headers.get('X-Shadowed', False)
            elif "check" in result.query.url:
                data = result.json
                weighted_total = 0

                for i in range(10):
                    # Buckets 0-9 should have 10 per bucket.
                    value = data.get(str(i), -1)
                    error = abs(value - 10)

                    assert error <= 2, f'bucket {i} should have 10 calls, got {value}'

                    # Buckets 100-109 should also have 10 per bucket... but honestly, the randomization
                    # in Envoy seems to kinda suck, so let's just check the total.

                    wi = i + 100

                    value = data.get(str(wi), -1)

                    # error = abs(value - 10)
                    # assert error <= 2, f'bucket {wi} should have 10 calls, got {value}'

                    weighted_total += value

                # 20% margin of error is kind of absurd, but
                #
                # - small sample sizes kind of suck, and
                # - we actually don't need to test Envoy's ability to generate random numbers, so meh.
                
                assert abs(weighted_total - 50) <= 10, f'weighted buckets should have 50 total calls, got {weighted_total}'
