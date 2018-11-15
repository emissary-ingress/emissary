import sys

from abc import ABC
from collections import OrderedDict
from typing import Any, Callable, Dict, List, Sequence, Tuple, Type

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

from multi import multi
from .parser import dump, load, Tag


def run(cmd):
    status = os.system(cmd)
    if status != 0:
        raise RuntimeError("command failed[%s]: %s" % (status, cmd))


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
        for ssc in get_nodes(sc):
            yield ssc


def variants(cls, *args, **kwargs) -> Tuple[Any]:
    return tuple(a for n in get_nodes(cls) for a in n.variants(*args, **kwargs))


class Name(str):

    @property
    def k8s(self):
        return self.replace(".", "-").lower()


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

    def __init__(self, *args, **kwargs) -> None:
        name = kwargs.pop("name", None)
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
            return Name(self.name)
        else:
            return Name(self.parent.relpath(ancestor) + "." + self.name)

    @property
    def k8s_path(self) -> str:
        return self.relpath(None).replace(".", "-").lower()

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


class Query:

    def __init__(self, url, expected=None, method="GET", headers=None, messages=None, insecure=False, skip=None,
                 xfail=None, phase=1, debug=False, sni=False, error=None):
        self.method = method
        self.url = url
        self.headers = headers
        self.messages = messages
        self.insecure = insecure
        if expected is None:
            if url.lower().startswith("ws:"):
                self.expected = 101
            else:
                self.expected = 200
        else:
            self.expected = expected
        self.skip = skip
        self.xfail = xfail
        self.phase = phase
        self.parent = None
        self.result = None
        self.debug = debug
        self.sni = sni
        self.error = error

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
        if self.headers:
            result["headers"] = self.headers
        if self.messages is not None:
            result["messages"] = self.messages
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

        if self.query.error is not None:
            assert self.query.error == self.error, "{}: expected error to be {}, got {} instead".format(
                self.query.url, self.query.error, self.error
            )
        else:
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

        if self.backend:
            od['backend'] = self.backend.as_dict()

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

    def as_dict(self) -> Dict[str, Any]:
        return {
            'enabled': self.enabled,
            'server_name': self.server_name,
            'version': self.version,
            'negotiated_protocol': self.negotiated_protocol,
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


CLIENT_GO = os.path.join(os.path.dirname(__file__), "client.go")


def run_queries(queries: Sequence[Query]) -> Sequence[Result]:
    jsonified = []
    byid = {}

    for q in queries:
        jsonified.append(q.as_json())
        byid[id(q)] = q

    with open("/tmp/urls.json", "w") as f:
        json.dump(jsonified, f)

    run("go run %s -input /tmp/urls.json -output /tmp/results.json 2> /tmp/client.log" % CLIENT_GO)

    with open("/tmp/results.json") as f:
        json_results = json.load(f)

    results = []

    for r in json_results:
        res = r["result"]
        q = byid[r["id"]]
        results.append(Result(q, res))

    return results


# yuck
DOCTEST = False


class Runner:

    def __init__(self, *classes, scope=None):
        self.scope = scope or "-".join(c.__name__ for c in classes)
        self.roots = tuple(v for c in classes for v in variants(c))
        self.nodes = [n for r in self.roots for n in r.traversal]
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

            expanded = set(selected)

            for e in list(expanded):
                for a in e.ancestors:
                    expanded.add(a)

            try:
                self._setup_k8s()

                for t in self.tests:
                    if t in expanded:
                        pre_query: Callable = getattr(t, "pre_query", None)

                        if pre_query:
                            pre_query()

                self._query(expanded)
            except:
                traceback.print_exc()
                pytest.exit("setup failed")
            finally:
                self.done = True

    def _setup_k8s(self):
        manifests = OrderedDict()
        for n in self.nodes:
            yaml = n.manifests()
            if yaml is not None:
                manifests[n] = load(n.path, yaml, Tag.MAPPING)

        configs = OrderedDict()
        for n in self.nodes:
            configs[n] = []
            for cfg in n.config():
                if isinstance(cfg, str):
                    parent_config = configs[n.parent][0][1][0]
                    for o in load(n.path, cfg, Tag.MAPPING):
                        parent_config.merge(o)
                else:
                    target = cfg[0]
                    yaml = load(n.path, cfg[1], Tag.MAPPING)

                    if n.ambassador_id:
                        for obj in yaml:
                            if "ambassador_id" not in obj:
                                obj["ambassador_id"] = n.ambassador_id

                    configs[n].append((target, yaml))

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

        if os.path.exists(fname):
            with open(fname) as f:
                prev_yaml = f.read()
        else:
            prev_yaml = None

        if yaml.strip() and (yaml != prev_yaml or DOCTEST):
            print("Manifests changed, applying.")
            with open(fname, "w") as f:
                f.write(yaml)
            # XXX: better prune selector label
            run("kubectl apply --prune -l scope=%s -f %s" % (self.scope, fname))
            self.applied_manifests = True
        elif yaml.strip():
            self.applied_manifests = False
            print("Manifests unchanged, skipping apply.")

        for n in self.nodes:
            action = getattr(n, "post_manifest", None)
            if action:
                action()

        self._wait()

    def _wait(self):
        requirements = [(node, kind, name) for node in self.nodes for kind, name in node.requirements()]

        homogenous = {}
        for node, kind, name in requirements:
            if kind not in homogenous:
                homogenous[kind] = []
            homogenous[kind].append((node, name))

        kinds = ["pod", "url"]
        delay = 0.5
        start = time.time()
        limit = 10*60

        while time.time() - start < limit:
            for kind in kinds:
                if kind not in homogenous:
                    continue

                reqs = homogenous[kind]

                print("Checking %s %s requirements... " % (len(reqs), kind), end="")

                sys.stdout.flush()

                if not self._ready(kind, reqs):
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

        assert False, "requirements not satisfied in %s seconds" % limit

    @multi
    def _ready(self, kind, _):
        return kind

    @_ready.when("pod")
    def _ready(self, _, requirements):
        pods = self._pods()
        for node, name in requirements:
            if not pods.get(name, False):
                print("%s not ready, " % name, end="")
                return False
        return True

    @_ready.when("url")
    def _ready(self, _, requirements):
        queries = []
        for node, q in requirements:
            q.insecure = True
            q.parent = node
            queries.append(q)

        result = run_queries(queries)

        not_ready = [r for r in result if r.status != r.query.expected]

        if not_ready:
            first = not_ready[0]
            print("%s not ready (%s) " % (first.query.url, first.status or first.error), end="")
            return False
        else:
            return True

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
            phase_queries = [q for q in queries if q.phase == phase]

            print("Querying %s urls in phase %s..." % (len(phase_queries), phase), end="")
            sys.stdout.flush()

            results = run_queries(phase_queries)

            print(" done.")

            for r in results:
                t = r.parent
                t.queried.append(r.query)

                if getattr(t, "debug", False) or getattr(r.query, "debug", False):
                    print("%s result: %s" % (t.name, json.dumps(r.as_dict(), sort_keys=True, indent=4)))

                t.results.append(r)
                t.pending.remove(r.query)
