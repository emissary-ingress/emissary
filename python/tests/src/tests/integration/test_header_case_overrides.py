import json
import logging
import sys
import time

import pytest
import requests

from tests.integration.utils import create_httpbin_mapping, install_ambassador
from tests.kubeutils import apply_kube_artifacts
from tests.manifests import httpbin_manifests
from tests.runutils import run_and_assert

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s test %(levelname)s: %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)

logger = logging.getLogger("ambassador")

from ambassador import IR, Config
from ambassador.envoy import EnvoyConfig
from ambassador.fetch import ResourceFetcher
from ambassador.utils import NullSecretHandler

headerecho_manifests = """
---
apiVersion: v1
kind: Service
metadata:
  name: headerecho
spec:
  type: ClusterIP
  selector:
    service: headerecho
  ports:
  - port: 80
    targetPort: http
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: headerecho
spec:
  replicas: 1
  selector:
    matchLabels:
      service: headerecho
  template:
    metadata:
      labels:
        service: headerecho
    spec:
      containers:
      - name: headerecho
        # We should find a better home for this image.
        image: johnesmet/simple-header-echo
        ports:
        - name: http
          containerPort: 8080
"""


def create_headerecho_mapping(namespace):
    headerecho_mapping = f"""
---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name:  headerecho-mapping
  namespace: {namespace}
spec:
  hostname: "*"
  prefix: /headerecho/
  rewrite: /
  service: headerecho
"""

    apply_kube_artifacts(namespace=namespace, artifacts=headerecho_mapping)


def _ambassador_module_config():
    return """
---
apiVersion: getambassador.io/v3alpha1
kind: Module
metadata:
  name: ambassador
  namespace: default
spec:
  config:
"""


def _ambassador_module_header_case_overrides(overrides, proper_case=False):
    mod = _ambassador_module_config()
    if len(overrides) == 0:
        mod = (
            mod
            + """
    header_case_overrides: []
"""
        )
        return mod

    mod = (
        mod
        + """
    header_case_overrides:
"""
    )
    for override in overrides:
        mod = (
            mod
            + f"""
    - {override}
"""
        )
    # proper case isn't valid if header_case_overrides are set, but we do
    # it here for tests that want to test that this is in fact invalid.
    if proper_case:
        mod = (
            mod
            + f"""
      proper_case: true
"""
        )
    return mod


def _test_headercaseoverrides(yaml, expectations, expect_norules=False):
    aconf = Config()

    yaml = (
        yaml
        + """
---
apiVersion: getambassador.io/v3alpha1
kind: Listener
metadata:
  name: ambassador-listener-8080
  namespace: default
spec:
  port: 8080
  protocol: HTTPS
  securityModel: XFP
  hostBinding:
    namespace:
      from: ALL

---
apiVersion: getambassador.io/v3alpha1
kind: Mapping
metadata:
  name: httpbin-mapping
  namespace: default
spec:
  service: httpbin
  hostname: "*"
  prefix: /httpbin/
"""
    )

    fetcher = ResourceFetcher(logger, aconf)
    fetcher.parse_yaml(yaml, k8s=True)

    aconf.load_all(fetcher.sorted())

    secret_handler = NullSecretHandler(logger, None, None, "0")

    ir = IR(aconf, file_checker=lambda path: True, secret_handler=secret_handler)
    assert ir

    econf = EnvoyConfig.generate(ir)
    assert econf, "could not create an econf"

    found_module_rules = False
    found_cluster_rules = False
    conf = econf.as_dict()

    for listener in conf["static_resources"]["listeners"]:
        for filter_chain in listener["filter_chains"]:
            for f in filter_chain["filters"]:
                typed_config = f["typed_config"]
                if "http_protocol_options" not in typed_config:
                    continue

                http_protocol_options = typed_config["http_protocol_options"]
                if expect_norules:
                    assert (
                        "header_key_format" not in http_protocol_options
                    ), f"'header_key_format' found unexpected typed_config {typed_config}"
                    continue

                assert (
                    "header_key_format" in http_protocol_options
                ), f"'header_key_format' not found, typed_config {typed_config}"

                header_key_format = http_protocol_options["header_key_format"]
                assert (
                    "custom" in header_key_format
                ), f"'custom' not found, typed_config {typed_config}"

                rules = header_key_format["custom"]["rules"]
                assert len(rules) == len(expectations)
                for e in expectations:
                    hdr = e.lower()
                    assert hdr in rules
                    rule = rules[hdr]
                    assert rule == e, f"unexpected rule {rule} in {rules}"
                found_module_rules = True

    for cluster in conf["static_resources"]["clusters"]:
        if "httpbin" not in cluster["name"]:
            continue

        http_protocol_options = cluster.get("http_protocol_options", None)
        if not http_protocol_options:
            if expect_norules:
                continue
            assert (
                "http_protocol_options" in cluster
            ), f"'http_protocol_options' missing from cluster: {cluster}"

        if expect_norules:
            assert (
                "header_key_format" not in http_protocol_options
            ), f"'header_key_format' found unexpected cluster: {cluster}"
            continue

        assert (
            "header_key_format" in http_protocol_options
        ), f"'header_key_format' not found, cluster {cluster}"

        header_key_format = http_protocol_options["header_key_format"]
        assert "custom" in header_key_format, f"'custom' not found, cluster {cluster}"

        rules = header_key_format["custom"]["rules"]
        assert len(rules) == len(expectations)
        for e in expectations:
            hdr = e.lower()
            assert hdr in rules
            rule = rules[hdr]
            assert rule == e, f"unexpected rule {rule} in {rules}"
        found_cluster_rules = True

    if expect_norules:
        assert not found_module_rules
        assert not found_cluster_rules
    else:
        assert found_module_rules
        assert found_cluster_rules


def _test_headercaseoverrides_rules(rules, expected=None, expect_norules=False):
    if not expected:
        expected = rules
    _test_headercaseoverrides(
        _ambassador_module_header_case_overrides(rules),
        expected,
        expect_norules=expect_norules,
    )


# Test that we throw assertions for obviously wrong cases
@pytest.mark.compilertest
def test_testsanity():
    failed = False
    try:
        _test_headercaseoverrides_rules(["X-ABC"], expected=["X-Wrong"])
    except AssertionError as e:
        failed = True
    assert failed

    failed = False
    try:
        _test_headercaseoverrides_rules([], expected=["X-Wrong"])
    except AssertionError as e:
        failed = True
    assert failed


# Test that we can parse a variety of header case override arrays.
@pytest.mark.compilertest
def test_headercaseoverrides_basic():
    _test_headercaseoverrides_rules([], expect_norules=True)
    _test_headercaseoverrides_rules([{}], expect_norules=True)
    _test_headercaseoverrides_rules([5], expect_norules=True)
    _test_headercaseoverrides_rules(["X-ABC"])
    _test_headercaseoverrides_rules(["X-foo", "X-ABC-Baz"])
    _test_headercaseoverrides_rules(["x-goOd", "X-alSo-good", "Authorization"])
    _test_headercaseoverrides_rules(["x-good", ["hello"]], expected=["x-good"])
    _test_headercaseoverrides_rules(["X-ABC", "x-foo", 5, {}], expected=["X-ABC", "x-foo"])


# Test that we always omit header case overrides if proper case is set
@pytest.mark.compilertest
def test_headercaseoverrides_propercasefail():
    _test_headercaseoverrides(
        _ambassador_module_header_case_overrides(["My-OPINIONATED-CASING"], proper_case=True),
        [],
        expect_norules=True,
    )
    _test_headercaseoverrides(
        _ambassador_module_header_case_overrides([], proper_case=True),
        [],
        expect_norules=True,
    )
    _test_headercaseoverrides(
        _ambassador_module_header_case_overrides([{"invalid": "true"}, "X-COOL"], proper_case=True),
        [],
        expect_norules=True,
    )


class HeaderCaseOverridesTesting:
    def create_module(self, namespace):
        manifest = f"""
---
apiVersion: getambassador.io/v3alpha1
kind: Module
metadata:
  name: ambassador
spec:
  config:
    header_case_overrides:
    - X-HELLO
    - X-FOO-Bar
        """

        apply_kube_artifacts(namespace=namespace, artifacts=manifest)

    def create_listeners(self, namespace):
        manifest = f"""
---
apiVersion: getambassador.io/v3alpha1
kind: Listener
metadata:
  name: listener-8080
spec:
  port: 8080
  protocol: HTTP
  securityModel: INSECURE
  hostBinding:
    namespace:
      from: SELF
"""

        apply_kube_artifacts(namespace=namespace, artifacts=manifest)

    def test_header_case_overrides(self):
        # Is there any reason not to use the default namespace?
        namespace = "header-case-overrides"

        # Install Ambassador
        install_ambassador(namespace=namespace)

        # Install httpbin
        apply_kube_artifacts(namespace=namespace, artifacts=httpbin_manifests)

        # Install headerecho
        apply_kube_artifacts(namespace=namespace, artifacts=headerecho_manifests)

        # Install listeners.
        self.create_listeners(namespace)

        # Install module
        self.create_module(namespace)

        # Install httpbin mapping
        create_httpbin_mapping(namespace=namespace)

        # Install headerecho mapping
        create_headerecho_mapping(namespace=namespace)

        # Now let's wait for ambassador and httpbin pods to become ready
        run_and_assert(
            [
                "tools/bin/kubectl",
                "wait",
                "--timeout=90s",
                "--for=condition=Ready",
                "pod",
                "-l",
                "service=ambassador",
                "-n",
                namespace,
            ]
        )
        run_and_assert(
            [
                "tools/bin/kubectl",
                "wait",
                "--timeout=90s",
                "--for=condition=Ready",
                "pod",
                "-l",
                "service=httpbin",
                "-n",
                namespace,
            ]
        )

        # Assume we can reach Ambassador through telepresence
        ambassador_host = "ambassador." + namespace

        # Assert 200 OK at httpbin/status/200 endpoint
        ready = False
        httpbin_url = f"http://{ambassador_host}/httpbin/status/200"
        headerecho_url = f"http://{ambassador_host}/headerecho/"

        loop_limit = 10
        while not ready:
            assert loop_limit > 0, "httpbin is not ready yet, aborting..."
            try:
                print(f"trying {httpbin_url}...")
                resp = requests.get(httpbin_url, timeout=5)
                code = resp.status_code
                assert code == 200, f"Expected 200 OK, got {code}"
                resp.close()
                print(f"{httpbin_url} is ready")

                print(f"trying {headerecho_url}...")
                resp = requests.get(headerecho_url, timeout=5)
                code = resp.status_code
                assert code == 200, f"Expected 200 OK, got {code}"
                resp.close()
                print(f"{headerecho_url} is ready")

                ready = True

            except Exception as e:
                print(f"Error: {e}")
                print(f"{httpbin_url} not ready yet, trying again...")
                time.sleep(1)
                loop_limit -= 1

        assert ready

        httpbin_url = f"http://{ambassador_host}/httpbin/response-headers?x-Hello=1&X-foo-Bar=1&x-Lowercase1=1&x-lowercase2=1"
        resp = requests.get(httpbin_url, timeout=5)
        code = resp.status_code
        assert code == 200, f"Expected 200 OK, got {code}"

        # First, test that the response headers have the correct case.

        # Very important: this test relies on matching case sensitive header keys.
        # Fortunately it appears that we can convert resp.headers, a case insensitive
        # dictionary, into a list of case sensitive keys.
        keys = [h for h in resp.headers.keys()]
        for k in keys:
            print(f"header key: {k}")

        assert "x-hello" not in keys
        assert "X-HELLO" in keys
        assert "x-foo-bar" not in keys
        assert "X-FOO-Bar" in keys
        assert "x-lowercase1" in keys
        assert "x-Lowercase1" not in keys
        assert "x-lowercase2" in keys
        resp.close()

        # Second, test that the request headers sent to the headerecho server
        # have the correct case.

        headers = {"x-Hello": "1", "X-foo-Bar": "1", "x-Lowercase1": "1", "x-lowercase2": "1"}
        resp = requests.get(headerecho_url, headers=headers, timeout=5)
        code = resp.status_code
        assert code == 200, f"Expected 200 OK, got {code}"

        response_obj = json.loads(resp.text)
        print(f"response_obj = {response_obj}")
        assert response_obj
        assert "headers" in response_obj

        hdrs = response_obj["headers"]
        assert "x-hello" not in hdrs
        assert "X-HELLO" in hdrs
        assert "x-foo-bar" not in hdrs
        assert "X-FOO-Bar" in hdrs
        assert "x-lowercase1" in hdrs
        assert "x-Lowercase1" not in hdrs
        assert "x-lowercase2" in hdrs


@pytest.mark.flaky(reruns=1, reruns_delay=10)
def test_ambassador_headercaseoverrides():
    t = HeaderCaseOverridesTesting()
    t.test_header_case_overrides()


if __name__ == "__main__":
    pytest.main(sys.argv)
