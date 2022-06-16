import subprocess
import sys

from abc import ABC
from collections import OrderedDict
from functools import singledispatch
from hashlib import sha256
from packaging import version
from typing import Any, Callable, Dict, List, Optional, Sequence, Tuple, Type, Union, cast

import base64
import fnmatch
import functools
import inspect
import json
import os
import pytest
import time
import threading
import traceback

from .utils import ShellCommand
from ambassador.utils import parse_bool

from yaml.scanner import ScannerError as YAMLScanError

import tests.integration.manifests as integration_manifests
from .parser import dump, load, Tag, SequenceView
from tests.manifests import httpbin_manifests, websocket_echo_server_manifests, cleartext_host_manifest, default_listener_manifest
from tests.kubeutils import apply_kube_artifacts

import yaml as pyyaml

pyyaml_loader: Any = pyyaml.SafeLoader
pyyaml_dumper: Any = pyyaml.SafeDumper

try:
    pyyaml_loader = pyyaml.CSafeLoader
    pyyaml_dumper = pyyaml.CSafeDumper
except AttributeError:
    pass

# Run mode can be local (don't do any Envoy stuff), envoy (only do Envoy stuff),
# or all (allow both). Default is all.
RUN_MODE = os.environ.get('KAT_RUN_MODE', 'all').lower()

# We may have a SOURCE_ROOT override from the environment
SOURCE_ROOT = os.environ.get('SOURCE_ROOT', '')

# Figure out if we're running in Edge Stack or what.
if os.path.exists("/buildroot/apro.version"):
    # We let /buildroot/apro.version remain a source of truth to minimize the
    # chances that we break anything that currently uses the builder shell.
    EDGE_STACK = True
else:
    # If we do not see concrete evidence of running in an apro builder shell,
    # then try to decide if the user wants us to assume we're running Edge Stack
    # from an environment variable. And if that isn't set, just assume OSS.
    EDGE_STACK = parse_bool(os.environ.get('EDGE_STACK', 'false'))

if EDGE_STACK:
    # Hey look, we're running inside Edge Stack!
    print("RUNNING IN EDGE STACK")
    # SOURCE_ROOT is optional, and we assume that if it isn't set, the user is
    # running in a build shell and we should look for sources in the usual location.
    if not SOURCE_ROOT:
        SOURCE_ROOT = "/buildroot/apro"
    GOLD_ROOT = os.path.join(SOURCE_ROOT, "tests/pytest/gold")
else:
    # We're either not running in Edge Stack or we're not sure, so just assume OSS.
    print("RUNNING IN OSS")
    # SOURCE_ROOT is optional, and we assume that if it isn't set, the user is
    # running in a build shell and we should look for sources in the usual location.
    if not SOURCE_ROOT:
        SOURCE_ROOT = "/buildroot/ambassador"
    GOLD_ROOT = os.path.join(SOURCE_ROOT, "python/tests/gold")


def run(cmd):
    status = os.system(cmd)
    if status != 0:
        raise RuntimeError("command failed[%s]: %s" % (status, cmd))


def kube_version_json():
    result = subprocess.Popen('tools/bin/kubectl version -o json', stdout=subprocess.PIPE, shell=True)
    stdout, _ = result.communicate()
    return json.loads(stdout)


def strip_version(ver: str):
    """
    strip_version is needed to strip a major/minor version of non-standard symbols. For example, when working with GKE,
    `kubectl version` returns a minor version like '14+', which is not semver or any standard version, for that matter.
    So we handle exceptions like that here.
    :param ver: version string
    :return: stripped version
    """

    try:
        return int(ver)
    except ValueError as e:
        # GKE returns weird versions with '+' in the end
        if ver[-1] == '+':
            return int(ver[:-1])

        # If we still have not taken care of this, raise the error
        raise ValueError(e)


def kube_server_version(version_json=None):
    if not version_json:
        version_json = kube_version_json()

    server_json = version_json.get('serverVersion', {})

    if server_json:
        server_major = strip_version(server_json.get('major', None))
        server_minor = strip_version(server_json.get('minor', None))

        return f"{server_major}.{server_minor}"
    else:
        return None


def kube_client_version(version_json=None):
    if not version_json:
        version_json = kube_version_json()

    client_json = version_json.get('clientVersion', {})

    if client_json:
        client_major = strip_version(client_json.get('major', None))
        client_minor = strip_version(client_json.get('minor', None))

        return f"{client_major}.{client_minor}"
    else:
        return None


def is_kube_server_client_compatible(debug_desc: str, requested_server_version: str, requested_client_version: str) -> bool:
    is_cluster_compatible = True
    kube_json = kube_version_json()

    server_version = kube_server_version(kube_json)
    client_version = kube_client_version(kube_json)

    if server_version:
        if version.parse(server_version) < version.parse(requested_server_version):
            print(f"server version {server_version} is incompatible with {debug_desc}")
            is_cluster_compatible = False
        else:
            print(f"server version {server_version} is compatible with {debug_desc}")
    else:
        print("could not determine Kubernetes server version?")

    if client_version:
        if version.parse(client_version) < version.parse(requested_client_version):
            print(f"client version {client_version} is incompatible with {debug_desc}")
            is_cluster_compatible = False
        else:
            print(f"client version {client_version} is compatible with {debug_desc}")
    else:
        print("could not determine Kubernetes client version?")

    return is_cluster_compatible


def is_ingress_class_compatible() -> bool:
    return is_kube_server_client_compatible('IngressClass', '1.18', '1.14')


def is_knative_compatible() -> bool:
    # Skip KNative immediately for run_mode local.
    if RUN_MODE == 'local':
        return False

    return is_kube_server_client_compatible('Knative', '1.14', '1.14')


def get_digest(data: str) -> str:
    s = sha256()
    s.update(data.encode('utf-8'))
    return s.hexdigest()


def has_changed(data: str, path: str) -> Tuple[bool, str]:
    cur_size = len(data.strip()) if data else 0
    cur_hash = get_digest(data)

    # print(f'has_changed: data size {cur_size} - {cur_hash}')

    prev_data = None
    changed = True
    reason = f'no {path} present'

    if os.path.exists(path):
        with open(path) as f:
            prev_data = f.read()

    prev_size = len(prev_data.strip()) if prev_data else 0
    prev_hash = None

    if prev_data:
        prev_hash = get_digest(prev_data)

    # print(f'has_changed: prev_data size {prev_size} - {prev_hash}')

    if data:
        if data != prev_data:
            reason = f'different data in {path}'
        else:
            changed = False
            reason = f'same data in {path}'

        if changed:
            # print(f'has_changed: updating {path}')
            with open(path, "w") as f:
                f.write(data)

    # For now, we always have to reapply with split testing.
    if not changed:
        changed = True
        reason = 'always reapply for split test'

    return (changed, reason)


COUNTERS: Dict[Type, int] = {}

SANITIZATIONS = OrderedDict((
    ("://", "SCHEME"),
    (":", "COLON"),
    (" ", "SPACE"),
    ("/t", "TAB"),
    (".", "DOT"),
    ("?", "QMARK"),
    ("/", "SLASH"),
))


def sanitize(obj):
    if isinstance(obj, str):
        for k, v in SANITIZATIONS.items():
            if obj.startswith(k):
                obj = obj.replace(k, v + "-")
            elif obj.endswith(k):
                obj = obj.replace(k, "-" + v)
            else:
                obj = obj.replace(k, "-" + v + "-")
        return obj
    elif isinstance(obj, dict):
        if 'value' in obj:
            return obj['value']
        else:
            return "-".join("%s-%s" % (sanitize(k), sanitize(v)) for k, v in sorted(obj.items()))
    else:
        cls = obj.__class__
        count = COUNTERS.get(cls, 0)
        COUNTERS[cls] = count + 1
        if count == 0:
            return cls.__name__
        else:
            return "%s-%s" % (cls.__name__, count)


def abstract_test(cls: type):
    cls.abstract_test = True  # type: ignore
    return cls


def get_nodes(node_type: type):
    if not inspect.isabstract(node_type) and not node_type.__dict__.get("abstract_test", False):
        yield node_type
    for sc in node_type.__subclasses__():
        if not sc.__dict__.get("skip_variant", False):
            for ssc in get_nodes(sc):
                yield ssc


def variants(cls, *args, **kwargs) -> Tuple[Any]:
    return tuple(a for n in get_nodes(cls) for a in n.variants(*args, **kwargs))  # type: ignore


class Name(str):
    namespace: Optional[str]

    def __new__(cls, value, namespace: Optional[str]=None):
        s = super().__new__(cls, value)
        s.namespace = namespace
        return s

    @property
    def k8s(self):
        return self.replace(".", "-").lower()

    @property
    def fqdn(self):
        r = self.k8s

        if self.namespace and (self.namespace != 'default'):
            r += '.' + self.namespace

        return r

class NodeLocal(threading.local):
    current: Optional['Node']

    def __init__(self):
        self.current = None


_local = NodeLocal()


def _argprocess(o):
    if isinstance(o, Node):
        return o.clone()
    elif isinstance(o, tuple):
        return tuple(_argprocess(i) for i in o)
    elif isinstance(o, list):
        return [_argprocess(i) for i in o]
    elif isinstance(o, dict):
        return {_argprocess(k): _argprocess(v) for k, v in o.items()}
    else:
        return o


class Node(ABC):

    parent: Optional['Node']
    children: List['Node']
    name: Name
    ambassador_id: str
    namespace: str = None  # type: ignore
    is_ambassador = False
    local_result: Optional[Dict[str, str]] = None

    def __init__(self, *args, **kwargs) -> None:
        # If self.skip is set to true, this node is skipped
        self.skip_node = False
        self.xfail: Optional[str] = None

        name = kwargs.pop("name", None)

        if 'namespace' in kwargs:
            self.namespace = kwargs.pop('namespace', None)

        _clone: Node = kwargs.pop("_clone", None)

        if _clone:
            args = _clone._args  # type: ignore
            kwargs = _clone._kwargs  # type: ignore
            if name:
                name = Name("-".join((_clone.name, name)))
            else:
                name = _clone.name
            self._args = _clone._args  # type: ignore
            self._kwargs = _clone._kwargs  # type: ignore
        else:
            self._args = args
            self._kwargs = kwargs
            if name:
                name = Name("-".join((self.__class__.__name__, name)))
            else:
                name = Name(self.__class__.__name__)

        saved = _local.current
        self.parent = _local.current

        if not self.namespace:
            if self.parent and self.parent.namespace:
                # We have no namespace assigned, but our parent does have a namespace
                # defined. Copy the namespace down from our parent.
                self.namespace = self.parent.namespace
            else:
                self.namespace = "default"

        _local.current = self
        self.children = []
        if self.parent is not None:
            self.parent.children.append(self)
        try:
            init = getattr(self, "init", lambda *a, **kw: None)
            init(*_argprocess(args), **_argprocess(kwargs))
        finally:
            _local.current = saved

        self.name = Name(self.format(name or self.__class__.__name__))

        names = {}  # type: ignore
        for c in self.children:
            assert c.name not in names, ("test %s of type %s has duplicate children: %s of type %s, %s" %
                                         (self.name, self.__class__.__name__, c.name, c.__class__.__name__,
                                          names[c.name].__class__.__name__))
            names[c.name] = c

    def clone(self, name=None):
        return self.__class__(_clone=self, name=name)

    def find_local_result(self, stop_at_first_ambassador: bool=False) -> Optional[Dict[str, str]]:
        test_name = self.format('{self.path.k8s}')

        # print(f"{test_name} {type(self)} FIND_LOCAL_RESULT")

        end_result: Optional[Dict[str, str]] = None

        n: Optional[Node] = self

        while n:
            node_name = n.format('{self.path.k8s}')
            parent = n.parent
            parent_name = parent.format('{self.path.k8s}') if parent else "-none-"

            end_result = getattr(n, 'local_result', None)
            result_str = end_result['result'] if end_result else '-none-'
            # print(f"{test_name}: {'ambassador' if n.is_ambassador else 'node'} {node_name}, parent {parent_name}, local_result = {result_str}")

            if end_result is not None:
                break

            if n.is_ambassador and stop_at_first_ambassador:
                # This is an Ambassador: don't continue past it.
                break

            n = n.parent

        return end_result

    def check_local(self, gold_root: str, k8s_yaml_path: str) -> Tuple[bool, bool]:
        testname = self.format('{self.path.k8s}')

        if self.xfail:
            # XFail early -- but still return True, True so that we don't try to run Envoy on it.
            self.local_result = {
                'result': 'xfail',
                'reason': self.xfail
            }
            # print(f"==== XFAIL: {testname} local: {self.xfail}")
            return (True, True)

        if not self.is_ambassador:
            # print(f"{testname} ({type(self)}) is not an Ambassador")
            return (False, False)

        if not self.ambassador_id:
            print(f"{testname} ({type(self)}) is an Ambassador but has no ambassador_id?")
            return (False, False)

        ambassador_namespace = getattr(self, 'namespace', 'default')
        ambassador_single_namespace = getattr(self, 'single_namespace', False)

        no_local_mode: bool = getattr(self, 'no_local_mode', False)
        skip_local_reason: Optional[str] = getattr(self, 'skip_local_instead_of_xfail', None)

        # print(f"{testname}: ns {ambassador_namespace} ({'single' if ambassador_single_namespace else 'multi'})")

        gold_path = os.path.join(gold_root, testname)

        if os.path.isdir(gold_path) and not no_local_mode:
            # print(f"==== {testname} running locally from {gold_path}")

            # Yeah, I know, brutal hack.
            #
            # XXX (Flynn) This code isn't used and we don't know if it works. If you try
            # it, bring it up-to-date with the environment created in abstract_tests.py
            envstuff = ["env", f"AMBASSADOR_NAMESPACE={ambassador_namespace}"]

            cmd = ["mockery", "--debug", k8s_yaml_path,
                   "-w", "python /ambassador/watch_hook.py",
                   "--kat", self.ambassador_id,
                   "--diff", gold_path]

            if ambassador_single_namespace:
                envstuff.append("AMBASSADOR_SINGLE_NAMESPACE=yes")
                cmd += ["-n", ambassador_namespace]

            if not getattr(self, 'allow_edge_stack_redirect', False):
                envstuff.append("AMBASSADOR_NO_TLS_REDIRECT=yes")

            cmd = envstuff + cmd

            w = ShellCommand(*cmd)

            if w.status():
                print(f"==== GOOD: {testname} local against {gold_path}")
                self.local_result = {'result': "pass"}
            else:
                print(f"==== FAIL: {testname} local against {gold_path}")

                self.local_result = {
                    'result': 'fail',
                    'stdout': w.stdout,
                    'stderr': w.stderr
                }

            return (True, True)
        else:
            # If we have a local reason, has a parent already subsumed us?
            #
            # XXX The way KAT works, our parent will have always run earlier than us, so
            # it's not clear if we can ever not have been subsumed.

            if skip_local_reason:
                local_result = self.find_local_result()

                if local_result:
                    self.local_result = {
                        'result': 'skip',
                        'reason': f"subsumed by {skip_local_reason} -- {local_result['result']}"
                    }
                    # print(f"==== {self.local_result['result'].upper()} {testname} {self.local_result['reason']}")
                    return (True, True)

            # OK, we weren't already subsumed. If we're in local mode, we'll skip or xfail
            # depending on skip_local_reason.

            if RUN_MODE == "local":
                if skip_local_reason:
                    self.local_result = {
                        'result': 'skip',
                        # 'reason': f"subsumed by {skip_local_reason} without result in local mode"
                    }
                    print(f"==== {self.local_result['result'].upper()} {testname} {self.local_result['reason']}")
                    return (True, True)
                else:
                    # XFail -- but still return True, True so that we don't try to run Envoy on it.
                    self.local_result = {
                        'result': 'xfail',
                        'reason': f"missing local cache {gold_path}"
                    }
                    # print(f"==== {self.local_result['result'].upper()} {testname} {self.local_result['reason']}")
                    return (True, True)

            # If here, we're not in local mode. Allow Envoy to run.
            self.local_result = None
            # print(f"==== IGNORE {testname} no local cache")
            return (True, False)

    def has_local_result(self) -> bool:
        return bool(self.local_result)

    @classmethod
    def variants(cls):
        yield cls()

    @property
    def path(self) -> Name:
        return self.relpath(None)

    def relpath(self, ancestor):
        if self.parent is ancestor:
            return Name(self.name, namespace=self.namespace)
        else:
            assert self.parent
            return Name(self.parent.relpath(ancestor) + "." + self.name, namespace=self.namespace)

    @property
    def traversal(self):
        yield self
        for c in self.children:
            for d in c.traversal:
                yield d

    @property
    def ancestors(self):
        yield self
        if self.parent is not None:
            for a in self.parent.ancestors:
                yield a

    @property
    def depth(self):
        if self.parent is None:
            return 0
        else:
            return self.parent.depth + 1

    def format(self, st, **kwargs):
        return integration_manifests.format(st, self=self, **kwargs)

    def get_fqdn(self, name: str) -> str:
        if self.namespace and (self.namespace != 'default'):
            return f'{name}.{self.namespace}'
        else:
            return name

    @functools.lru_cache()
    def matches(self, pattern):
        if fnmatch.fnmatch(self.path, "*%s*" % pattern):
            return True
        for c in self.children:
            if c.matches(pattern):
                return True
        return False

    def requirements(self):
        yield from ()

    # log_kube_artifacts writes various logs about our underlying Kubernetes objects to
    # a place where the artifact publisher can find them. See run-tests.sh.
    def log_kube_artifacts(self):
        if not getattr(self, 'already_logged', False):
            self.already_logged = True

            print(f'logging kube artifacts for {self.path.k8s}')
            sys.stdout.flush()

            DEV = os.environ.get("AMBASSADOR_DEV", "0").lower() in ("1", "yes", "true")

            log_path = f'/tmp/kat-logs-{self.path.k8s}'

            if DEV:
                os.system(f'docker logs {self.path.k8s} >{log_path} 2>&1')
            else:
                os.system(f'tools/bin/kubectl logs -n {self.namespace} {self.path.k8s} >{log_path} 2>&1')

                event_path = f'/tmp/kat-events-{self.path.k8s}'

                fs1 = f'involvedObject.name={self.path.k8s}'
                fs2 = f'involvedObject.namespace={self.namespace}'

                cmd = f'tools/bin/kubectl get events -o json --field-selector "{fs1}" --field-selector "{fs2}"'
                os.system(f'echo ==== "{cmd}" >{event_path}')
                os.system(f'{cmd} >>{event_path} 2>&1')


class Test(Node):

    results: Sequence['Result']

    __test__ = False

    def config(self):
        yield from ()

    def manifests(self):
        return None

    def queries(self):
        yield from ()

    def check(self):
        pass

    def handle_local_result(self) -> bool:
        test_name = self.format('{self.path.k8s}')

        # print(f"{test_name} {type(self)} HANDLE_LOCAL_RESULT")

        end_result = self.find_local_result()

        if end_result is not None:
            result_type = end_result['result']

            if result_type == 'pass':
                pass
            elif result_type == 'skip':
                pytest.skip(end_result['reason'])
            elif result_type == 'fail':
                sys.stdout.write(end_result['stdout'])

                if os.environ.get('KAT_VERBOSE', None):
                    sys.stderr.write(end_result['stderr'])

                pytest.fail("local check failed")
            elif result_type == 'xfail':
                pytest.xfail(end_result['reason'])

            return True

        return False

    @property
    def ambassador_id(self):
        if self.parent is None:
            return self.name.k8s
        else:
            return self.parent.ambassador_id


@singledispatch
def encode_body(obj):
    return encode_body(json.dumps(obj))

@encode_body.register
def encode_body_bytes(b: bytes):
    return base64.encodebytes(b).decode("utf-8")

@encode_body.register
def encode_body_str(s: str):
    return encode_body(s.encode("utf-8"))

class Query:

    def __init__(self, url, expected=None, method="GET", headers=None, messages=None, insecure=False, skip=None,
                 xfail=None, phase=1, debug=False, sni=False, error=None, client_crt=None, client_key=None,
                 client_cert_required=False, ca_cert=None, grpc_type=None, cookies=None, ignore_result=False, body=None,
                 minTLSv="", maxTLSv="", cipherSuites=[], ecdhCurves=[]):
        self.method = method
        self.url = url
        self.headers = headers
        self.body = body
        self.cookies = cookies
        self.messages = messages
        self.insecure = insecure
        self.minTLSv = minTLSv
        self.maxTLSv = maxTLSv
        self.cipherSuites = cipherSuites
        self.ecdhCurves = ecdhCurves
        if expected is None:
            if url.lower().startswith("ws:"):
                self.expected = 101
            else:
                self.expected = 200
        else:
            self.expected = expected
        self.skip = skip
        self.xfail = xfail
        self.ignore_result = ignore_result
        self.phase = phase
        self.parent = None
        self.result = None
        self.debug = debug
        self.sni = sni
        self.error = error
        self.client_cert_required = client_cert_required
        self.client_cert = client_crt
        self.client_key = client_key
        self.ca_cert = ca_cert
        assert grpc_type in (None, "real", "bridge", "web"), grpc_type
        self.grpc_type = grpc_type

    def as_json(self):
        assert self.parent
        result = {
            "test": self.parent.path,
            "id": id(self),
            "url": self.url,
            "insecure": self.insecure
        }
        if self.sni:
            result["sni"] = self.sni
        if self.method:
            result["method"] = self.method
        if self.method:
            result["maxTLSv"] = self.maxTLSv
        if self.method:
            result["minTLSv"] = self.minTLSv
        if self.cipherSuites:
            result["cipherSuites"] = self.cipherSuites
        if self.ecdhCurves:
            result["ecdhCurves"] = self.ecdhCurves
        if self.headers:
            result["headers"] = self.headers
        if self.body is not None:
            result["body"] = encode_body(self.body)
        if self.cookies:
            result["cookies"] = self.cookies
        if self.messages is not None:
            result["messages"] = self.messages
        if self.client_cert is not None:
            result["client_cert"] = self.client_cert
        if self.client_key is not None:
            result["client_key"] = self.client_key
        if self.ca_cert is not None:
            result["ca_cert"] = self.ca_cert
        if self.client_cert_required:
            result["client_cert_required"] = self.client_cert_required
        if self.grpc_type:
            result["grpc_type"] = self.grpc_type

        return result


class Result:
    body: Optional[bytes]

    def __init__(self, query, res):
        self.query = query
        query.result = self
        self.parent = query.parent
        self.status = res.get("status")
        self.headers = res.get("headers")
        self.messages = res.get("messages")
        self.tls = res.get("tls")
        if "body" in res:
            self.body = base64.decodebytes(bytes(res["body"], "ASCII"))
        else:
            self.body = None
        self.text = res.get("text")
        self.json = res.get("json")
        self.backend = BackendResult(self.json) if self.json else None
        self.error = res.get("error")

    def __repr__(self):
        return str(self.as_dict())

    def check(self):
        if self.query.skip:
            pytest.skip(self.query.skip)

        if self.query.xfail:
            pytest.xfail(self.query.xfail)

        if not self.query.ignore_result:
            if self.query.error is not None:
                found = False
                errors = self.query.error

                if isinstance(self.query.error, str):
                    errors = [ self.query.error ]

                if self.error is not None:
                    for error in errors:
                        if error in self.error:
                            found = True
                            break

                assert found, "{}: expected error to contain any of {}; got {} instead".format(
                    self.query.url, ", ".join([ "'%s'" % x for x in errors ]),
                    ("'%s'" % self.error) if self.error else "no error"
                )
            else:
                if self.query.expected != self.status:
                    self.parent.log_kube_artifacts()

                assert self.query.expected == self.status, \
                       "%s: expected status code %s, got %s instead with error %s" % (
                           self.query.url, self.query.expected, self.status, self.error)

    def as_dict(self) -> Dict[str, Any]:
        od = {
            'query': self.query.as_json(),
            'status': self.status,
            'error': self.error,
            'headers': self.headers,
        }

        if self.backend and self.backend.name:
            od['backend'] = self.backend.as_dict()
        else:
            od['json'] = self.json
            od['text'] = self.text

        return od

        # 'RENDERED': {
        #     'client': {
        #         'request': self.query.as_json(),
        #         'response': {
        #             'status': self.status,
        #             'error': self.error,
        #             'headers': self.headers
        #         }
        #     },
        #     'upstream': {
        #         'name': self.backend.name,
        #         'request': {
        #             'headers': self.backend.request.headers,
        #             'url': {
        #                 'fragment': self.backend.request.url.fragment,
        #                 'host': self.backend.request.url.host,
        #                 'opaque': self.backend.request.url.opaque,
        #                 'path': self.backend.request.url.path,
        #                 'query': self.backend.request.url.query,
        #                 'rawQuery': self.backend.request.url.rawQuery,
        #                 'scheme': self.backend.request.url.scheme,
        #                 'username': self.backend.request.url.username,
        #                 'password': self.backend.request.url.password,
        #             },
        #             'host': self.backend.request.host,
        #             'tls': {
        #                 'enabled': self.backend.request.tls.enabled,
        #                 'server_name': self.backend.request.tls.server_name,
        #                 'version': self.backend.request.tls.version,
        #                 'negotiated_protocol': self.backend.request.tls.negotiated_protocol,
        #             },
        #         },
        #         'response': {
        #             'headers': self.backend.response.headers
        #         }
        #     }
        # }


class BackendURL:

    def __init__(self, fragment=None, host=None, opaque=None, path=None, query=None, rawQuery=None,
                 scheme=None, username=None, password=None):
        self.fragment = fragment
        self.host = host
        self.opaque = opaque
        self.path = path
        self.query = query
        self.rawQuery = rawQuery
        self.scheme = scheme
        self.username = username
        self.password = password

    def as_dict(self) -> Dict['str', Any]:
        return {
            'fragment': self.fragment,
            'host': self.host,
            'opaque': self.opaque,
            'path': self.path,
            'query': self.query,
            'rawQuery': self.rawQuery,
            'scheme': self.scheme,
            'username': self.username,
            'password': self.password,
        }


class BackendRequest:

    def __init__(self, req):
        self.url = BackendURL(**req.get("url"))
        self.headers = req.get("headers", {})
        self.host = req.get("host", None)
        self.tls = BackendTLS(req.get("tls", {}))

    def as_dict(self) -> Dict[str, Any]:
        od = {
            'headers': self.headers,
            'host': self.host,
        }

        if self.url:
            od['url'] = self.url.as_dict()

        if self.tls:
            od['tls'] = self.tls.as_dict()

        return od


class BackendTLS:

    def __init__(self, tls):
        self.enabled = tls["enabled"]
        self.server_name = tls.get("server-name")
        self.version = tls.get("version")
        self.negotiated_protocol = tls.get("negotiated-protocol")
        self.negotiated_protocol_version = tls.get("negotiated-protocol-version")

    def as_dict(self) -> Dict[str, Any]:
        return {
            'enabled': self.enabled,
            'server_name': self.server_name,
            'version': self.version,
            'negotiated_protocol': self.negotiated_protocol,
            'negotiated_protocol_version': self.negotiated_protocol_version,
        }


class BackendResponse:

    def __init__(self, resp):
        self.headers = resp.get("headers", {})

    def as_dict(self) -> Dict[str, Any]:
        return { 'headers': self.headers }


def dictify(obj):
    if getattr(obj, "as_dict", None):
        return obj.as_dict()
    else:
        return obj


class BackendResult:

    def __init__(self, bres):
        self.name = "raw"
        self.request = None
        self.response = bres

        if isinstance(bres, dict):
            self.name = cast(str, bres.get("backend"))
            self.request = BackendRequest(bres["request"]) if "request" in bres else None
            self.response = BackendResponse(bres["response"]) if "response" in bres else None

    def as_dict(self) -> Dict[str, Any]:
        od = {
            'name': self.name
        }

        if self.request:
            od['request'] = dictify(self.request)

        if self.response:
            od['response'] = dictify(self.response)

        return od


def label(yaml, scope):
    for obj in yaml:
        md = obj["metadata"]

        if "labels" not in md:
            md["labels"] = {}

        obj["metadata"]["labels"]["scope"] = scope
    return yaml


CLIENT_GO = "kat_client"

def run_queries(name: str, queries: Sequence[Query]) -> Sequence[Result]:
    jsonified = []
    byid = {}

    for q in queries:
        jsonified.append(q.as_json())
        byid[id(q)] = q

    path_urls = f'/tmp/kat-client-{name}-urls.json'
    path_results = f'/tmp/kat-client-{name}-results.json'
    path_log = f'/tmp/kat-client-{name}.log'

    with open(path_urls, 'w') as f:
        json.dump(jsonified, f)

    # run(f"{CLIENT_GO} -input {path_urls} -output {path_results} 2> {path_log}")
    res = ShellCommand.run('Running queries',
            f"tools/bin/kubectl exec -n default -i kat /work/kat_client < '{path_urls}' > '{path_results}' 2> '{path_log}'",
            shell=True)

    if not res:
        ret = [Result(q, {"error":"Command execution error"}) for q in queries]
        return ret

    with open(path_results, 'r') as f:
        content = f.read()
        try:
            json_results = json.loads(content)
        except Exception as e:
            ret = [Result(q, {"error":"Could not parse JSON content after running KAT queries"}) for q in queries]
            return ret

    results = []

    for r in json_results:
        res = r["result"]
        q = byid[r["id"]]
        results.append(Result(q, res))

    return results


# yuck
DOCTEST = False


class Superpod:
    def __init__(self, namespace: str) -> None:
        self.namespace = namespace
        self.next_clear = 8080
        self.next_tls = 8443
        self.service_names: Dict[int, str] = {}
        self.name = 'superpod-%s' % (self.namespace or 'default')

    def allocate(self, service_name) -> List[int]:
        ports = [ self.next_clear, self.next_tls ]
        self.service_names[self.next_clear] = service_name
        self.service_names[self.next_tls] = service_name

        self.next_clear += 1
        self.next_tls += 1

        return ports

    def get_manifest_list(self) -> List[Dict[str, Any]]:
        manifest = load('superpod', integration_manifests.format(integration_manifests.load("superpod_pod")), Tag.MAPPING)

        assert len(manifest) == 1, "SUPERPOD manifest must have exactly one object"

        m = manifest[0]

        template = m['spec']['template']

        ports: List[Dict[str, int]] = []
        envs: List[Dict[str, Union[str, int]]] = template['spec']['containers'][0]['env']

        for p in sorted(self.service_names.keys()):
            ports.append({ 'containerPort': p })
            envs.append({ 'name': f'BACKEND_{p}', 'value': self.service_names[p] })

        template['spec']['containers'][0]['ports'] = ports

        if 'metadata' not in m:
            m['metadata'] = {}

        metadata = m['metadata']
        metadata['name'] = self.name

        m['spec']['selector']['matchLabels']['backend'] = self.name
        template['metadata']['labels']['backend'] = self.name

        if self.namespace:
            # Fix up the namespace.
            if 'namespace' not in metadata:
                metadata['namespace'] = self.namespace

        return list(manifest)


class Runner:

    def __init__(self, *classes, scope=None):
        self.scope = scope or "-".join(c.__name__ for c in classes)
        self.roots = tuple(v for c in classes for v in variants(c))
        self.nodes = [n for r in self.roots for n in r.traversal if not n.skip_node]
        self.tests = [n for n in self.nodes if isinstance(n, Test)]
        self.ids = [t.path for t in self.tests]
        self.done = False
        self.skip_nonlocal_tests = False
        self.ids_to_strip: Dict[str, bool] = {}
        self.names_to_ignore: Dict[str, bool] = {}

        @pytest.mark.parametrize("t", self.tests, ids=self.ids)
        def test(request, capsys, t):
            if t.xfail:
                pytest.xfail(t.xfail)
            else:
                selected = set(item.callspec.getparam('t') for item in request.session.items if item.function == test)

                with capsys.disabled():
                    self.setup(selected)

                if not t.handle_local_result():
                    # XXX: should aggregate the result of url checks
                    i = 0
                    for r in t.results:
                        try:
                            r.check()
                        except AssertionError as e:
                            # Add some context so that you can tell which query is failing.
                            e.args = (f"query[{i}]: {e.args[0]}", *e.args[1:])
                            raise
                        i += 1

                    t.check()

        self.__func__ = test
        self.__test__ = True

    def __call__(self):
        assert False, "this is here for py.test discovery purposes only"

    def setup(self, selected):
        if not self.done:
            if not DOCTEST:
                print()

            expanded_up = set(selected)

            for s in selected:
                for n in s.ancestors:
                    if not n.xfail:
                        expanded_up.add(n)

            expanded = set(expanded_up)

            for s in selected:
                for n in s.traversal:
                    if not n.xfail:
                        expanded.add(n)

            try:
                self._setup_k8s(expanded)

                if self.skip_nonlocal_tests:
                    self.done = True
                    return

                for t in self.tests:
                    if t.has_local_result():
                        # print(f"{t.name}: SKIP due to local result")
                        continue

                    if t in expanded_up:
                        pre_query: Callable = getattr(t, "pre_query", None)

                        if pre_query:
                            pre_query()

                self._query(expanded_up)
            except:
                traceback.print_exc()
                pytest.exit("setup failed")
            finally:
                self.done = True

    def get_manifests_and_namespaces(self, selected) -> Tuple[Any, List[str]]:
        manifests: OrderedDict[Any, list] = OrderedDict()  # type: ignore
        superpods: Dict[str, Superpod] = {}
        namespaces = []
        for n in (n for n in self.nodes if n in selected and not n.xfail):
            manifest = None
            nsp = None
            ambassador_id = None

            # print('manifesting for {n.path}')

            # Walk up the parent chain to find our namespace and ambassador_id.
            cur = n

            while cur:
                if not nsp:
                    nsp = getattr(cur, 'namespace', None)
                    # print(f'... {cur.name} has namespace {nsp}')

                if not ambassador_id:
                    ambassador_id = getattr(cur, 'ambassador_id', None)
                    # print(f'... {cur.name} has ambassador_id {ambassador_id}')

                if nsp and ambassador_id:
                    # print(f'... good for namespace and ambassador_id')
                    break

                cur = cur.parent

            # OK. Does this node want to use a superpod?
            if getattr(n, 'use_superpod', False):
                # Yup. OK. Do we already have a superpod for this namespace?
                superpod = superpods.get(nsp, None)  # type: ignore

                if not superpod:
                    # We don't have one, so we need to create one.
                    superpod = Superpod(nsp)  # type: ignore
                    superpods[nsp] = superpod  # type: ignore

                # print(f'superpodifying {n.name}')

                # Next up: use the backend_service.yaml manifest as a template...
                yaml = n.format(integration_manifests.load("backend_service"))
                manifest = load(n.path, yaml, Tag.MAPPING)

                assert len(manifest) == 1, "backend_service.yaml manifest must have exactly one object"

                m = manifest[0]

                # Update the manifest's selector...
                m['spec']['selector']['backend'] = superpod.name

                # ...and labels if needed...
                if ambassador_id:
                    m['metadata']['labels'] = { 'kat-ambassador-id': ambassador_id }

                # ...and target ports.
                superpod_ports = superpod.allocate(n.path.k8s)

                m['spec']['ports'][0]['targetPort'] = superpod_ports[0]
                m['spec']['ports'][1]['targetPort'] = superpod_ports[1]
            else:
                # The non-superpod case...
                yaml = n.manifests()

                if yaml is not None:
                    is_plain_test = n.path.k8s.startswith("plain-")

                    if n.is_ambassador and not is_plain_test:
                        add_default_http_listener = getattr(n, 'add_default_http_listener', True)
                        add_default_https_listener = getattr(n, 'add_default_https_listener', True)
                        add_cleartext_host = getattr(n, 'edge_stack_cleartext_host', False)

                        if add_default_http_listener:
                            # print(f"{n.path.k8s} adding default HTTP Listener")
                            yaml += default_listener_manifest % {
                                "namespace": nsp,
                                "port": 8080,
                                "protocol": "HTTPS",
                                "securityModel": "XFP"
                            }

                        if add_default_https_listener:
                            # print(f"{n.path.k8s} adding default HTTPS Listener")
                            yaml += default_listener_manifest % {
                                "namespace": nsp,
                                "port": 8443,
                                "protocol": "HTTPS",
                                "securityModel": "XFP"
                            }

                        if EDGE_STACK and add_cleartext_host:
                            # print(f"{n.path.k8s} adding Host")

                            host_yaml = cleartext_host_manifest % nsp
                            yaml += host_yaml

                    yaml = n.format(yaml)

                    try:
                        manifest = load(n.path, yaml, Tag.MAPPING)
                    except Exception as e:
                        print(f'parse failure! {e}')
                        print(yaml)

            if manifest:
                # print(manifest)

                # Make sure namespaces and labels are properly set.
                for m in manifest:
                    if 'metadata' not in m:
                        m['metadata'] = {}

                    metadata = m['metadata']

                    if 'labels' not in metadata:
                        metadata['labels'] = {}

                    if ambassador_id:
                        metadata['labels']['kat-ambassador-id'] = ambassador_id

                    if nsp:
                        if 'namespace' not in metadata:
                            metadata['namespace'] = nsp

                # ...and, finally, save the manifest list.
                manifests[n] = list(manifest)
                if str(nsp) not in namespaces:
                    namespaces.append(str(nsp))

        for superpod in superpods.values():
            manifests[superpod] = superpod.get_manifest_list()

        return manifests, namespaces

    def do_local_checks(self, selected, fname) -> bool:
        if RUN_MODE == 'envoy':
            print("Local mode not allowed, continuing to Envoy mode")
            return False

        all_valid = True
        self.ids_to_strip = {}
        # This feels a bit wrong?
        self.names_to_ignore = {}

        for n in (n for n in self.nodes if n in selected):
            local_possible, local_checked = n.check_local(GOLD_ROOT, fname)

            if local_possible:
                if local_checked:
                    self.ids_to_strip[n.ambassador_id] = True
                else:
                    all_valid = False

        return all_valid

    def _setup_k8s(self, selected):
        # First up, get the full manifest and save it to disk.
        manifests, namespaces = self.get_manifests_and_namespaces(selected)

        configs: Dict[Node, List[Tuple[str, SequenceView]]] = OrderedDict()
        for n in (n for n in self.nodes if n in selected and not n.xfail):
            configs[n] = []
            for cfg in n.config():
                if isinstance(cfg, str):
                    parent_config = configs[n.parent][0][1][0]

                    try:
                        for o in load(n.path, cfg, Tag.MAPPING):
                            parent_config.merge(o)
                    except YAMLScanError as e:
                        raise Exception("Parse Error: %s, input text:\n%s" % (e, cfg))
                else:
                    target = cfg[0]

                    try:
                        yaml_view = load(n.path, cfg[1], Tag.MAPPING)

                        if n.ambassador_id:
                            for obj in yaml_view:
                                if "ambassador_id" not in obj:
                                    obj["ambassador_id"] = [n.ambassador_id]

                        configs[n].append((target, yaml_view))
                    except YAMLScanError as e:
                        raise Exception("Parse Error: %s, input text:\n%s" % (e, cfg[1]))

        for tgt_cfgs in configs.values():
            for target, cfg in tgt_cfgs:
                for t in target.traversal:
                    if t in manifests:
                        k8s_yaml = manifests[t]
                        for item in k8s_yaml:
                            if item["kind"].lower() == "service":
                                md = item["metadata"]
                                if "annotations" not in md:
                                    md["annotations"] = {}

                                anns = md["annotations"]

                                if "getambassador.io/config" in anns:
                                    anns["getambassador.io/config"] += "\n" + dump(cfg)
                                else:
                                    anns["getambassador.io/config"] = dump(cfg)

                                break
                        else:
                            continue
                        break
                else:
                    assert False, "no service found for target: %s" % target.path

        yaml = ""

        for v in manifests.values():
            yaml += dump(label(v, self.scope)) + "\n"

        fname = "/tmp/k8s-%s.yaml" % self.scope

        self.applied_manifests = False

        # Always apply at this point, since we're doing the multi-run thing.
        manifest_changed, manifest_reason = has_changed(yaml, fname)

        # OK. Try running local stuff.
        if self.do_local_checks(selected, fname):
            # Everything that could run locally did. Good enough.
            self.skip_nonlocal_tests = True
            return True

        # Something didn't work out quite right.
        print(f'Continuing with Kube tests...')
        # print(f"ids_to_strip {self.ids_to_strip}")

        # XXX It is _so stupid_ that we're reparsing the whole manifest here.
        xxx_crap = pyyaml.load_all(open(fname, "r").read(), Loader=pyyaml_loader)

        # Strip things we don't need from the manifest.
        trimmed_manifests = []
        trimmed = 0
        kept = 0

        for obj in xxx_crap:
            keep = True

            kind = '-nokind-'
            name = '-noname-'
            metadata: Dict[str, Any] = {}
            labels: Dict[str, str] = {}
            id_to_check: Optional[str] = None

            if 'kind' in obj:
                kind = obj['kind']

            if 'metadata' in obj:
                metadata = obj['metadata']

            if 'name' in metadata:
                name = metadata['name']

            if 'labels' in metadata:
                labels = metadata['labels']

            if 'kat-ambassador-id' in labels:
                id_to_check = labels['kat-ambassador-id']

            # print(f"metadata {metadata} id_to_check {id_to_check} obj {obj}")

            # Keep namespaces, just in case.
            if kind == 'Namespace':
                keep = True
            else:
                if id_to_check and (id_to_check in self.ids_to_strip):
                    keep = False
                    # print(f"...drop {kind} {name} (ID {id_to_check})")
                    self.names_to_ignore[name] = True

            if keep:
                kept += 1
                trimmed_manifests.append(obj)
            else:
                trimmed += 1

        if trimmed:
            print(f"After trimming: kept {kept}, trimmed {trimmed}")

        yaml = pyyaml.dump_all(trimmed_manifests, Dumper=pyyaml_dumper)

        fname = "/tmp/k8s-%s-trimmed.yaml" % self.scope

        self.applied_manifests = False

        # Always apply at this point, since we're doing the multi-run thing.
        manifest_changed, manifest_reason = has_changed(yaml, fname)

        # First up: CRDs.
        input_crds = integration_manifests.crd_manifests()
        if is_knative_compatible():
            input_crds += integration_manifests.load("knative_serving_crds")

        # Strip out all of the schema validation, so that we can test with broken CRDs.
        # (KAT isn't really in the business of testing to be sure that Kubernetes can
        # run the K8s validators...)
        crds = pyyaml.load_all(input_crds, Loader=pyyaml_loader)

        # Collect the CRDs with schema validation stripped in stripped_crds, because
        # pyyaml.load_all actually returns something more complex than a simple list,
        # so it doesn't reserialize well after being modified.
        stripped_crds = []

        for crd in crds:
            # Guard against empty CRDs (the KNative files have some blank lines at
            # the end).
            if not crd:
                continue

            if crd["apiVersion"] == "apiextensions.k8s.io/v1":
                # We can't naively strip the schema validation from apiextensions.k8s.io/v1 CRDs
                # because it is required; otherwise the API server would refuse to create the CRD,
                # telling us:
                #
                #     CustomResourceDefinition.apiextensions.k8s.io "…" is invalid: spec.versions[0].schema.openAPIV3Schema: Required value: schemas are required
                #
                # So instead we must replace it with a schema that allows anything.
                for version in crd["spec"]["versions"]:
                    if "schema" in version:
                        version["schema"] = {
                            'openAPIV3Schema': {
                                'type': 'object',
                                'properties': {
                                    'apiVersion': { 'type': 'string' },
                                    'kind':       { 'type': 'string' },
                                    'metadata':   { 'type': 'object' },
                                    'spec': {
                                        'type': 'object',
                                        'x-kubernetes-preserve-unknown-fields': True,
                                    },
                                },
                            },
                        }
            elif crd["apiVersion"] == "apiextensions.k8s.io/v1beta1":
                crd["spec"].pop("validation", None)
                for version in crd["spec"]["versions"]:
                    version.pop("schema", None)
            stripped_crds.append(crd)

        final_crds = pyyaml.dump_all(stripped_crds, Dumper=pyyaml_dumper)
        changed, reason = has_changed(final_crds, "/tmp/k8s-CRDs.yaml")

        if changed:
            print(f'CRDS changed ({reason}), applying.')
            if not ShellCommand.run_with_retry(
                    'Apply CRDs',
                    'tools/bin/kubectl', 'apply', '-f', '/tmp/k8s-CRDs.yaml',
                    retries=5, sleep_seconds=10):
                raise RuntimeError("Failed applying CRDs")

            tries_left = 10

            while os.system('tools/bin/kubectl get crd mappings.getambassador.io > /dev/null 2>&1') != 0:
                tries_left -= 1

                if tries_left <= 0:
                    raise RuntimeError("CRDs never became available")

                print("sleeping for CRDs... (%d)" % tries_left)
                time.sleep(5)
        else:
            print(f'CRDS unchanged {reason}, skipping apply.')

        # Next up: the KAT pod.
        kat_client_manifests = integration_manifests.load("kat_client_pod")
        if os.environ.get("DEV_USE_IMAGEPULLSECRET", False):
            kat_client_manifests = integration_manifests.namespace_manifest("default") + kat_client_manifests
        changed, reason = has_changed(integration_manifests.format(kat_client_manifests), "/tmp/k8s-kat-pod.yaml")

        if changed:
            print(f'KAT pod definition changed ({reason}), applying')
            if not ShellCommand.run_with_retry('Apply KAT pod',
                    'tools/bin/kubectl', 'apply', '-f' , '/tmp/k8s-kat-pod.yaml', '-n', 'default',
                    retries=5, sleep_seconds=10):
                raise RuntimeError('Could not apply manifest for KAT pod')

            tries_left = 3
            time.sleep(1)

            while True:
                if ShellCommand.run("wait for KAT pod",
                                    'tools/bin/kubectl', '-n', 'default', 'wait', '--timeout=30s', '--for=condition=Ready', 'pod', 'kat'):
                    print("KAT pod ready")
                    break

                tries_left -= 1

                if tries_left <= 0:
                    raise RuntimeError("KAT pod never became available")

                print("sleeping for KAT pod... (%d)" % tries_left)
                time.sleep(5)
        else:
            print(f'KAT pod definition unchanged {reason}, skipping apply.')

        # Use a dummy pod to get around the !*@&#$!*@&# DockerHub rate limit.
        # XXX Better: switch to GCR.
        dummy_pod = integration_manifests.load("dummy_pod")
        if os.environ.get("DEV_USE_IMAGEPULLSECRET", False):
            dummy_pod = integration_manifests.namespace_manifest("default") + dummy_pod
        changed, reason = has_changed(integration_manifests.format(dummy_pod), "/tmp/k8s-dummy-pod.yaml")

        if changed:
            print(f'Dummy pod definition changed ({reason}), applying')
            if not ShellCommand.run_with_retry('Apply dummy pod',
                    'tools/bin/kubectl', 'apply', '-f' , '/tmp/k8s-dummy-pod.yaml', '-n', 'default',
                    retries=5, sleep_seconds=10):
                raise RuntimeError('Could not apply manifest for dummy pod')

            tries_left = 3
            time.sleep(1)

            while True:
                if ShellCommand.run("wait for dummy pod",
                                    'tools/bin/kubectl', '-n', 'default', 'wait', '--timeout=30s', '--for=condition=Ready', 'pod', 'dummy-pod'):
                    print("Dummy pod ready")
                    break

                tries_left -= 1

                if tries_left <= 0:
                    raise RuntimeError("Dummy pod never became available")

                print("sleeping for dummy pod... (%d)" % tries_left)
                time.sleep(5)
        else:
            print(f'Dummy pod definition unchanged {reason}, skipping apply.')

        # # Clear out old stuff.
        if os.environ.get("DEV_CLEAN_K8S_RESOURCES", False):
            print("Clearing cluster...")
            ShellCommand.run('clear old Kubernetes namespaces',
                             'tools/bin/kubectl', 'delete', 'namespaces', '-l', 'scope=AmbassadorTest',
                             verbose=True)
            ShellCommand.run('clear old Kubernetes pods etc.',
                             'tools/bin/kubectl', 'delete', 'all', '-l', 'scope=AmbassadorTest', '--all-namespaces',
                             verbose=True)

        # XXX: better prune selector label
        if manifest_changed:
            print(f"manifest changed ({manifest_reason}), applying...")
            if not ShellCommand.run_with_retry('Applying k8s manifests',
                    'tools/bin/kubectl', 'apply', '--prune', '-l', 'scope=%s' % self.scope, '-f', fname,
                    retries=5, sleep_seconds=10):
                raise RuntimeError('Could not apply manifests')
            self.applied_manifests = True

        # Finally, install httpbin and the websocket-echo-server.
        print(f"applying http_manifests + websocket_echo_server_manifests to namespaces: {namespaces}")
        for namespace in namespaces:
            apply_kube_artifacts(namespace, httpbin_manifests)
            apply_kube_artifacts(namespace, websocket_echo_server_manifests)

        for n in self.nodes:
            if n in selected and not n.xfail:
                action = getattr(n, "post_manifest", None)
                if action:
                    action()

        self._wait(selected)

        print("Waiting 5s after requirements, just because...")
        time.sleep(5)

    @staticmethod
    def _req_str(kind, req) -> str:
        printable = req

        if kind == 'url':
            printable = req.url

        return printable

    def _wait(self, selected: Sequence[Node]):
        requirements: List[Tuple[Node, str, Query]] = []

        for node in selected:
            if node.xfail:
                continue

            node_name = node.format("{self.path.k8s}")
            ambassador_id = getattr(node, 'ambassador_id', None)

            # print(f"{node_name} {ambassador_id}")

            if node.has_local_result():
                # print(f"{node_name} has local result, skipping")
                continue

            if ambassador_id and ambassador_id in self.ids_to_strip:
                # print(f"{node_name} has id {ambassador_id}, stripping")
                continue

            if node_name in self.names_to_ignore:
                # print(f"{node_name} marked to ignore, stripping")
                continue

            # if RUN_MODE != "envoy":
            #     print(f"{node_name}: including in nonlocal tests")

            for kind, req in node.requirements():
                # print(f"{node_name} add req ({node_name}, {kind}, {self._req_str(kind, req)})")
                requirements.append((node, kind, req))

        homogenous: Dict[str, List[Tuple[Node, Query]]] = {}

        for node, kind, name in requirements:
            if kind not in homogenous:
                homogenous[kind] = []

            homogenous[kind].append((node, name))

        kinds = [ "pod", "url" ]
        delay = 5
        start = time.time()
        limit = int(os.environ.get("KAT_REQ_LIMIT", "900"))

        print("Starting requirements check (limit %ds)... " % limit)

        holdouts = {}

        while time.time() - start < limit:
            for kind in kinds:
                if kind not in homogenous:
                    continue

                reqs = homogenous[kind]

                print("Checking %s %s requirements... " % (len(reqs), kind), end="")

                # print("\n")
                # for node, req in reqs:
                #     print(f"...{node.format('{self.path.k8s}')} - {self._req_str(kind, req)}")

                sys.stdout.flush()

                is_ready, _holdouts = self._ready(kind, reqs)

                if not is_ready:
                    holdouts[kind] = _holdouts
                    delay = int(min(delay*2, 10))
                    print("sleeping %ss..." % delay)
                    sys.stdout.flush()
                    time.sleep(delay)
                else:
                    print("satisfied.")
                    sys.stdout.flush()
                    kinds.remove(kind)

                break
            else:
                return

        print("requirements not satisfied in %s seconds:" % limit)

        for kind in kinds:
            _holdouts = holdouts.get(kind, [])

            if _holdouts:
                print(f'  {kind}:')

                for node, text in _holdouts:
                    print(f'    {node.path.k8s} ({text})')
                    node.log_kube_artifacts()

        assert False, "requirements not satisfied in %s seconds" % limit

    def _ready(self, kind, requirements):
        fn = {
            "pod": self._ready_pod,
            "url": self._ready_url,
        }[kind]

        return fn(kind, requirements)

    def _ready_pod(self, _, requirements):
        pods = self._pods(self.scope)
        not_ready = []

        for node, name in requirements:
            if not pods.get(name, False):
                not_ready.append((node, name))

        if not_ready:
            print("%d not ready (%s), " % (len(not_ready), name), end="")
            return (False, not_ready)

        return (True, None)

    def _ready_url(self, _, requirements):
        queries = []

        for node, q in requirements:
            q.insecure = True
            q.parent = node
            queries.append(q)

        # print("URL Reqs:")
        # print("\n".join([ f'{q.parent.name}: {q.url}' for q in queries ]))

        result = run_queries("reqcheck", queries)

        not_ready = [r for r in result if r.status != r.query.expected]

        if not_ready:
            first = not_ready[0]
            print("%d not ready (%s: %s) " % (len(not_ready), first.query.url, first.status or first.error), end="")
            return (False, [ (x.query.parent, "%s -- %s" % (x.query.url, x.status or x.error)) for x in not_ready ])
        else:
            return (True, None)

    def _pods(self, scope=None):
        scope_for_path = scope if scope else 'global'
        label_for_scope = f'-l scope={scope}' if scope else ''

        fname = f'/tmp/pods-{scope_for_path}.json'
        if not ShellCommand.run_with_retry('Getting pods',
            f'tools/bin/kubectl get pod {label_for_scope} --all-namespaces -o json > {fname}',
            shell=True, retries=5, sleep_seconds=10):
            raise RuntimeError('Could not get pods')


        with open(fname) as f:
            raw_pods = json.load(f)

        pods = {}

        for p in raw_pods["items"]:
            name = p["metadata"]["name"]

            cstats = p["status"].get("containerStatuses", [])

            all_ready = True

            for status in cstats:
                ready = status.get('ready', False)

                if not ready:
                    all_ready = False
                    # print(f'pod {name} is not ready: {status.get("state", "unknown state")}')

            pods[name] = all_ready

        return pods

    def _query(self, selected) -> None:
        queries = []

        for t in self.tests:
            t_name = t.format('{self.path.k8s}')

            if t in selected:
                t.pending = []
                t.queried = []
                t.results = []
            else:
                continue

            if t.has_local_result():
                # print(f"{t_name}: SKIP QUERY due to local result")
                continue

            ambassador_id = getattr(t, 'ambassador_id', None)

            if ambassador_id and ambassador_id in self.ids_to_strip:
                # print(f"{t_name}: SKIP QUERY due to ambassador_id {ambassador_id}")
                continue

            # print(f"{t_name}: INCLUDE QUERY")
            for q in t.queries():
                q.parent = t
                t.pending.append(q)
                queries.append(q)

        phases = sorted(set([q.phase for q in queries]))
        first = True

        for phase in phases:
            if not first:
                phase_delay = int(os.environ.get("KAT_PHASE_DELAY", 10))
                print("Waiting for {} seconds before starting phase {}...".format(phase_delay, phase))
                time.sleep(phase_delay)

            first = False

            phase_queries = [q for q in queries if q.phase == phase]

            print("Querying %s urls in phase %s..." % (len(phase_queries), phase), end="")
            sys.stdout.flush()

            results = run_queries(f'phase{phase}', phase_queries)

            print(" done.")

            for r in results:
                t = r.parent
                t.queried.append(r.query)

                if getattr(t, "debug", False) or getattr(r.query, "debug", False):
                    print("%s result: %s" % (t.name, json.dumps(r.as_dict(), sort_keys=True, indent=4)))

                t.results.append(r)
                t.pending.remove(r.query)
