from abc import ABC, abstractmethod
from collections import OrderedDict
from itertools import chain, product
from typing import Any, Iterable, Mapping, Optional, Sequence, Tuple, Type

import base64, copy, fnmatch, functools, inspect, json, os, pprint, pytest, sys, time, threading, traceback

from .parser import dump, load, Tag

def run(cmd):
    status = os.system(cmd)
    if status != 0:
        raise RuntimeError("command failed[%s]: %s" % (status, cmd))

COUNTERS: Mapping[Type,int] = {}

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
            return cls.__name__ + "-%s" %  count

def abstract_test(cls):
    cls.abstract_test = True
    return cls

def get_nodes(type):
    if not inspect.isabstract(type) and not type.__dict__.get("abstract_test", False):
        yield type
    for sc in type.__subclasses__():
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

    parent: 'Test'
    children: Sequence['Test']
    name: str

    def __init__(self, *args, **kwargs):
        name = kwargs.pop("name", None)
        _clone = kwargs.pop("_clone", None)
        if _clone:
            args = _clone._args
            kwargs = _clone._kwargs
            if name:
                name = "-".join((_clone.name, name))
            else:
                name = _clone.name
            self._args = _clone._args
            self._kwargs = _clone._kwargs
        else:
            self._args = args
            self._kwargs = kwargs
            if name:
                name = "-".join((self.__class__.__name__, name))
            else:
                name = self.__class__.__name__

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
        if False: yield

class Test(Node):

    __test__ = False

    def config(self):
        if False: yield

    def manifests(self):
        return None

    def queries(self):
        if False: yield

    def check(self):
        pass

    @property
    def ambassador_id(self):
        if self.parent is None:
            return self.name.k8s
        else:
            return self.parent.ambassador_id

class Query:

    def __init__(self, url, expected=200, method="GET", headers=None, insecure=False, skip = None, xfail = None):
        self.method = method
        self.url = url
        self.headers = headers
        self.insecure = insecure
        self.expected = expected
        self.skip = skip
        self.xfail = xfail
        self.parent = None
        self.result = None

    def as_json(self):
        result = {
            "test": self.parent.path, "id": id(self),
            "url": self.url,
            "insecure": self.insecure
        }
        if self.method:
            result["method"] = self.method
        if self.headers:
            result["headers"] = self.headers
        return result

class Result:

    def __init__(self, query, res):
        self.query = query
        query.result = self
        self.parent = query.parent
        self.status = res.get("status")
        self.headers = res.get("headers")
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
        assert self.query.expected == self.status, "%s: expected %s, got %s" % (self.query.url, self.query.expected, self.status or self.error)

    def as_dict(self):
        return {
            'RENDERED': {
                'client': {
                    'request': self.query.as_json(),
                    'response': {
                        'status': self.status,
                        'error': self.error,
                        'headers': self.headers
                    }
                },
                'upstream': {
                    'name': self.backend.name,
                    'request': {
                        'headers': self.backend.request.headers,
                        'url': {
                            'fragment': self.backend.request.url.fragment,
                            'host': self.backend.request.url.host,
                            'opaque': self.backend.request.url.opaque,
                            'path': self.backend.request.url.path,
                            'query': self.backend.request.url.query,
                            'rawQuery': self.backend.request.url.rawQuery,
                            'scheme': self.backend.request.url.scheme,
                            'username': self.backend.request.url.username,
                            'password': self.backend.request.url.password,
                        },
                        'host': self.backend.request.host,
                        'tls': {
                            'enabled': self.backend.request.tls.enabled,
                            'server_name': self.backend.request.tls.server_name,
                            'version': self.backend.request.tls.version,
                            'negotiated_protocol': self.backend.request.tls.negotiated_protocol,
                        },
                    },
                    'response': {
                        'headers': self.backend.response.headers
                    }
                }
            },
            'ACTUAL': {
                'query': self.query.as_json(),
                'status': self.status,
                'error': self.error,
                'headers': self.headers,
                'backend': {
                    'name': self.backend.name,
                    'request': {
                        'headers': self.backend.request.headers,
                        'url': {
                            'fragment': self.backend.request.url.fragment,
                            'host': self.backend.request.url.host,
                            'opaque': self.backend.request.url.opaque,
                            'path': self.backend.request.url.path,
                            'query': self.backend.request.url.query,
                            'rawQuery': self.backend.request.url.rawQuery,
                            'scheme': self.backend.request.url.scheme,
                            'username': self.backend.request.url.username,
                            'password': self.backend.request.url.password,
                        },
                        'host': self.backend.request.host,
                        'tls': {
                            'enabled': self.backend.request.tls.enabled,
                            'server_name': self.backend.request.tls.server_name,
                            'version': self.backend.request.tls.version,
                            'negotiated_protocol': self.backend.request.tls.negotiated_protocol,
                        }
                    },
                    'response': {
                        'headers': self.backend.response.headers
                    }
                },
            }
        }

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

class BackendRequest:

    def __init__(self, req):
        self.url = BackendURL(**req.get("url"))
        self.headers = req.get("headers", {})
        self.host = req.get("host", None)
        self.tls = BackendTLS(req.get("tls", {}))

class BackendTLS:

    def __init__(self, tls):
        self.enabled = tls["enabled"]
        self.server_name = tls.get("server-name")
        self.version = tls.get("version")
        self.negotiated_protocol = tls.get("negotiated-protocol")

class BackendResponse:

    def __init__(self, resp):
        self.headers = resp.get("headers", {})

class BackendResult:

    def __init__(self, bres):
        self.name = bres.get("backend")
        self.request = BackendRequest(bres["request"]) if "request" in bres else None
        self.response = BackendResponse(bres["response"]) if "response" in bres else None

def label(yaml, scope):
    for obj in yaml:
        md = obj["metadata"]
        if "labels" not in md: md["labels"] = {}
        obj["metadata"]["labels"]["scope"] = scope
    return yaml


CLIENT_GO = os.path.join(os.path.dirname(__file__), "client.go")

def query(queries: Sequence[Query]) -> Sequence[Result]:
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

    def run(self):
        for t in self.tests:
            try:
                self.setup(set(self.tests))
                for r in t.results:
                    r.check()
                t.check()
                print("%s: PASSED" % t.name)
            except:
                print("%s: FAILED\n  %s" % (t.name, traceback.format_exc().replace("\n", "\n  ")))

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
                    if t in expanded and getattr(t, "pre_query", None):
                        t.pre_query()
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
                    if n.ambassador_id != None:
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
            self._wait()
        elif yaml.strip():
            print("Manifests unchanged, skipping apply.")

    def _wait(self):
        requirements = [r for n in self.nodes for r in n.requirements()]
        if not requirements: return

        for i in range(30):
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

            print("Checking requirements... ", end="")
            sys.stdout.flush()
            for kind, name in requirements:
                assert kind == "pod"
                if not pods.get(name, False):
                    print("%s %s not ready, sleeping..." % (kind, name))
                    sys.stdout.flush()
                    time.sleep(10)
                    break
            else:
                print("satisfied.")
                return

        assert False, "requirements not satisfied within 5 minutes"

    def _query(self, selected):
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

        if queries:
            print("Querying %s urls..." % len(queries), end="")
            sys.stdout.flush()
            results = query(queries)
            print(" done.")

            for r in results:
                t = r.parent
                t.queried.append(r.query)

                if getattr(t, "debug", False):
                    print("%s result: %s" % (t.name, json.dumps(r.as_dict(), sort_keys=True, indent=4)))

                t.results.append(r)
                t.pending.remove(r.query)
