import json
import pytest

from typing import ClassVar, Dict, List, Sequence, Tuple, Union

from kat.harness import sanitize, variants, Query, Runner
from kat import manifests

from abstract_tests import AmbassadorTest, HTTP
from abstract_tests import MappingTest, OptionTest, ServiceType, Node, Test


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
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.name}-target
prefix: /{self.name}/mark/
rewrite: /mark/
service: https://{self.target.path.fqdn}
---
apiVersion: ambassador/v0
kind:  Mapping
name:  {self.name}-shadow
prefix: /{self.name}/mark/
rewrite: /mark/
service: shadow.plain-namespace
shadow: true
---
apiVersion: ambassador/v0
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
        for i in range(100):
            yield Query(self.parent.url("%s/mark/%d" % (self.name, i % 10)))

        yield Query(self.parent.url("%s/check/" % self.name), phase=2)

    def check(self):
        for result in self.results:
            if "mark" in result.query.url:
                assert not result.headers.get('X-Shadowed', False)
            elif "check" in result.query.url:
                data = result.json
                errors = 0

                for i in range(10):
                    value = data.get(str(i), -1)
                    error = abs(value - 10)

                    if error > 2:
                        errors += 1

                assert errors == 0
