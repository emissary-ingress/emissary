import subprocess
import sys

from abc import ABC
from collections import OrderedDict
from hashlib import sha256
from typing import Any, Callable, Dict, List, Sequence, Tuple, Type, Union
from packaging import version

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

from .manifests import BACKEND_SERVICE, SUPERPOD_POD, CRDS, KNATIVE_SERVING_CRDS

from yaml.scanner import ScannerError as YAMLScanError

from multi import multi
from .parser import dump, load, Tag


def run(cmd):
    status = os.system(cmd)
    if status != 0:
        raise RuntimeError("command failed[%s]: %s" % (status, cmd))


def kube_version_json():
    result = subprocess.Popen('kubectl version -o json', stdout=subprocess.PIPE, shell=True)
    stdout, _ = result.communicate()
    return json.loads(stdout)


def kube_server_version():
    version_json = kube_version_json()
    server_json = version_json['serverVersion']
    return f"{server_json['major']}.{server_json['minor']}"


def kube_client_version():
    version_json = kube_version_json()
    client_json = version_json['clientVersion']
    return f"{client_json['major']}.{client_json['minor']}"


def is_knative():
    is_cluster_compatible = True
    server_version = kube_server_version()
    client_version = kube_client_version()
    if version.parse(server_version) < version.parse('1.11'):
        print(f"server version {server_version} is incompatible with Knative")
        is_cluster_compatible = False
    else:
        print(f"server version {server_version} is compatible with Knative")

    if version.parse(client_version) < version.parse('1.10'):
        print(f"client version {client_version} is incompatible with Knative")
        is_cluster_compatible = False
    else:
        print(f"client version {client_version} is compatible with Knative")

    return is_cluster_compatible


def get_digest(data: str) -> str:
    s = sha256()
    s.update(data.encode('utf-8'))
    return s.hexdigest()


def has_changed(data: str, path: str) -> Tuple[bool, str]:
    cur_size = len(data.strip()) if data else 0
    cur_hash = get_digest(data)

    print(f'has_changed: data size {cur_size} - {cur_hash}')

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

    print(f'has_changed: prev_data size {prev_size} - {prev_hash}')

    if data:
        if data != prev_data:
            reason = f'different data in {path}'
        else:
            changed = False
            reason = f'same data in {path}'

        if changed:
            print(f'has_changed: updating {path}')
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
    cls.abstract_test = True
    return cls


def get_nodes(node_type: type):
    if not inspect.isabstract(node_type) and not node_type.__dict__.get("abstract_test", False):
        yield node_type
    for sc in node_type.__subclasses__():
        if not sc.__dict__.get("skip_variant", False):
            for ssc in get_nodes(sc):
                yield ssc


def variants(cls, *args, **kwargs) -> Tuple[Any]:
    return tuple(a for n in get_nodes(cls) for a in n.variants(*args, **kwargs))


class Name(str):
    def __new__(cls, value, namespace=None):
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

    parent: 'Node'
    children: List['Node']
    name: Name
    ambassador_id: str
    namespace: str = None

    def __init__(self, *args, **kwargs) -> None:
        # If self.skip is set to true, this node is skipped
        self.skip_node = False

        name = kwargs.pop("name", None)

        if 'namespace' in kwargs:
            self.namespace = kwargs.pop('namespace', None)

        _clone: Node = kwargs.pop("_clone", None)

        if _clone:
            args = _clone._args
            kwargs = _clone._kwargs
            if name:
                name = Name("-".join((_clone.name, name)))
            else:
                name = _clone.name
            self._args = _clone._args
            self._kwargs = _clone._kwargs
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

        names = {}
        for c in self.children:
            assert c.name not in names, ("test %s of type %s has duplicate children: %s of type %s, %s" %
                                         (self.name, self.__class__.__name__, c.name, c.__class__.__name__,
                                          names[c.name].__class__.__name__))
            names[c.name] = c

    def clone(self, name=None):
        return self.__class__(_clone=self, name=name)

    @classmethod
    def variants(cls):
        yield cls()

    @property
    def path(self) -> str:
        return self.relpath(None)

    def relpath(self, ancestor):
        if self.parent is ancestor:
            return Name(self.name, namespace=self.namespace)
        else:
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
        return st.format(self=self, **kwargs)

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
                os.system(f'kubectl logs -n {self.namespace} {self.path.k8s} >{log_path} 2>&1')

                event_path = f'/tmp/kat-events-{self.path.k8s}'

                fs1 = f'involvedObject.name={self.path.k8s}'
                fs2 = f'involvedObject.namespace={self.namespace}'

                cmd = f'kubectl get events -o json --field-selector "{fs1}" --field-selector "{fs2}"'
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

    @property
    def ambassador_id(self):
        if self.parent is None:
            return self.name.k8s
        else:
            return self.parent.ambassador_id


@multi
def encode_body(obj):
    yield type(obj)

@encode_body.when(bytes)
def encode_body(b):
    return base64.encodebytes(b).decode("utf-8")

@encode_body.when(str)
def encode_body(s):
    return encode_body(s.encode("utf-8"))

@encode_body.default
def encode_body(obj):
    return encode_body(json.dumps(obj))

class Query:

    def __init__(self, url, expected=None, method="GET", headers=None, messages=None, insecure=False, skip=None,
                 xfail=None, phase=1, debug=False, sni=False, error=None, client_crt=None, client_key=None,
                 client_cert_required=False, ca_cert=None, grpc_type=None, cookies=None, ignore_result=False, body=None,
                 minTLSv="", maxTLSv=""):
        self.method = method
        self.url = url
        self.headers = headers
        self.body = body
        self.cookies = cookies
        self.messages = messages
        self.insecure = insecure
        self.minTLSv = minTLSv
        self.maxTLSv = maxTLSv
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
        result = {
            "test": self.parent.path, "id": id(self),
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
            self.name = bres.get("backend")
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

    run(f"{CLIENT_GO} -input {path_urls} -output {path_results} 2> {path_log}")

    with open(path_results, 'r') as f:
        json_results = json.load(f)

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
        manifest = load('superpod', SUPERPOD_POD, Tag.MAPPING)

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

        @pytest.mark.parametrize("t", self.tests, ids=self.ids)
        def test(request, capsys, t):
            selected = set(item.callspec.getparam('t') for item in request.session.items if item.function == test)

            with capsys.disabled():
                self.setup(selected)

            # XXX: should aggregate the result of url checks
            for r in t.results:
                r.check()

            t.check()

        self.__func__ = test
        self.__test__ = True

    def __call__(self):
        assert False, "this is here for py.test discovery purposes only"

    # def run(self):
    #     for t in self.tests:
    #         try:
    #             self.setup(set(self.tests))
    #
    #             for r in t.results:
    #                 print("%s - %s: checking (2)" % (r.parent.name, r.query.url))
    #
    #                 r.check()
    #
    #                 if r.query.expected != r.status:
    #                     print("%s - %s: failed (2)" % (r.parent.name, r.query.url))
    #                     assert (False,
    #                             "%s: expected %s, got %s" % (r.query.url, r.query.expected, r.status or r.error))
    #
    #             t.check()
    #
    #             print("%s: PASSED" % t.name)
    #         except:
    #             print("%s: FAILED\n  %s" % (t.name, traceback.format_exc().replace("\n", "\n  ")))

    def setup(self, selected):
        if not self.done:
            if not DOCTEST:
                print()

            expanded_up = set(selected)

            for s in selected:
                for n in s.ancestors:
                    expanded_up.add(n)

            expanded = set(expanded_up)

            for s in selected:
                for n in s.traversal:
                    expanded.add(n)

            try:
                self._setup_k8s(expanded)

                for t in self.tests:
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

    def get_manifests(self, selected) -> OrderedDict:
        manifests = OrderedDict()
        superpods: Dict[str, Superpod] = {}

        for n in (n for n in self.nodes if n in selected):
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
                superpod = superpods.get(nsp, None)

                if not superpod:
                    # We don't have one, so we need to create one.
                    superpod = Superpod(nsp)
                    superpods[nsp] = superpod

                # print(f'superpodifying {n.name}')

                # Next up: use the BACKEND_SERVICE manifest as a template...
                yaml = n.format(BACKEND_SERVICE)
                manifest = load(n.path, yaml, Tag.MAPPING)

                assert len(manifest) == 1, "BACKEND_SERVICE manifest must have exactly one object"

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
                    manifest = load(n.path, yaml, Tag.MAPPING)

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
                manifests[n] = manifest

        for superpod in superpods.values():
            manifests[superpod] = superpod.get_manifest_list()

        return manifests

    def _setup_k8s(self, selected):
        # First up: CRDs.
        final_crds = CRDS
        if is_knative():
            final_crds += KNATIVE_SERVING_CRDS

        changed, reason = has_changed(final_crds, "/tmp/k8s-CRDs.yaml")

        if changed:
            print(f'CRDS changed ({reason}), applying.')
            run(f'kubectl apply -f /tmp/k8s-CRDs.yaml')

            tries_left = 10

            while os.system('kubectl get crd mappings.getambassador.io > /dev/null 2>&1') != 0:
                tries_left -= 1

                if tries_left <= 0:
                    raise RuntimeError("CRDs never became available")

                print("sleeping for CRDs... (%d)" % tries_left)
                time.sleep(5)
        else:
            print(f'CRDS unchanged {reason}, skipping apply.')

        manifests = self.get_manifests(selected)

        configs = OrderedDict()
        for n in (n for n in self.nodes if n in selected):
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
                        yaml = load(n.path, cfg[1], Tag.MAPPING)

                        if n.ambassador_id:
                            for obj in yaml:
                                if "ambassador_id" not in obj:
                                    obj["ambassador_id"] = n.ambassador_id

                        configs[n].append((target, yaml))
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

        # # Clear out old stuff.
        # print("Clearing cluster...")
        # ShellCommand.run('clear old Kubernetes namespaces',
        #                  'kubectl', 'delete', 'namespaces', '-l', 'scope=AmbassadorTest',
        #                  verbose=True)
        # ShellCommand.run('clear old Kubernetes pods etc.',
        #                  'kubectl', 'delete', 'all', '-l', 'scope=AmbassadorTest', '--all-namespaces',
        #                  verbose=True)

        self.applied_manifests = False

        # Always apply at this point, since we're doing the multi-run thing.
        changed, reason = has_changed(yaml, fname)

        if changed:
            print(f'Manifests changed ({reason}), applying.')

            # XXX: better prune selector label
            run("kubectl apply --prune -l scope=%s -f %s" % (self.scope, fname))
            self.applied_manifests = True
        else:
            print(f'Manifests unchanged ({reason}), applying.')

        for n in self.nodes:
            if n in selected:
                action = getattr(n, "post_manifest", None)
                if action:
                    action()

        self._wait(selected)

    def _wait(self, selected):
        requirements = [ (node, kind, name) for node in self.nodes for kind, name in node.requirements()
                         if node in selected ]

        homogenous = {}

        for node, kind, name in requirements:
            if kind not in homogenous:
                homogenous[kind] = []

            homogenous[kind].append((node, name))

        kinds = [ "pod", "url" ]
        delay = 5
        start = time.time()
        limit = int(os.environ.get("KAT_REQ_LIMIT", "300"))

        print("Starting requirements check (limit %ds)... " % limit)

        holdouts = {}

        while time.time() - start < limit:
            for kind in kinds:
                if kind not in homogenous:
                    continue

                reqs = homogenous[kind]

                print("Checking %s %s requirements... " % (len(reqs), kind), end="")

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

    @multi
    def _ready(self, kind, _):
        return kind

    @_ready.when("pod")
    def _ready(self, _, requirements):
        pods = self._pods()
        not_ready = []

        for node, name in requirements:
            if not pods.get(name, False):
                not_ready.append((node, name))

        if not_ready:
            print("%d not ready (%s), " % (len(not_ready), name), end="")
            return (False, not_ready)

        return (True, None)

    @_ready.when("url")
    def _ready(self, _, requirements):
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

    def _pods(self):
        fname = "/tmp/pods-%s.json" % self.scope
        run("kubectl get pod -l scope=%s -o json > %s" % (self.scope, fname))

        with open(fname) as f:
            raw_pods = json.load(f)

        pods = {}

        for p in raw_pods["items"]:
            name = p["metadata"]["name"]
            statuses = tuple(cs["ready"] for cs in p["status"].get("containerStatuses", ()))

            if not statuses:
                ready = False
            else:
                ready = True

                for status in statuses:
                    ready = ready and status

            pods[name] = ready

        return pods

    def _query(self, selected) -> None:
        queries = []

        for t in self.tests:
            if t in selected:
                t.pending = []
                t.queried = []
                t.results = []

                for q in t.queries():
                    q.parent = t
                    t.pending.append(q)
                    queries.append(q)

        phases = sorted(set([q.phase for q in queries]))

        for phase in phases:
            if phase != 1:
                phase_delay = int(os.environ.get("KAT_PHASE_DELAY", 10))
                print("Waiting for {} seconds before starting phase {}...".format(phase_delay, phase))
                time.sleep(phase_delay)

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
