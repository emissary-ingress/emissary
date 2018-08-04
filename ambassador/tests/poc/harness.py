from abc import ABC, abstractmethod
from itertools import chain, product
from typing import Any, Iterable, Mapping, Optional, Sequence, Type

import base64, copy, fnmatch, functools, inspect, json, os, pprint, pytest, sys

COUNTERS: Mapping[Type,int] = {}

SANITIZATIONS = {
    " ": "SPACE",
    "/t": "TAB",
    ".": "DOT",
    "?": "QMARK",
    "/": "SLASH"
}

def sanitize(obj):
    if isinstance(obj, str):
        for k, v in SANITIZATIONS.items():
            if obj.startswith(k):
                obj = obj.replace(k, v + "-")
            elif obj.endswith(k):
                obj = obj.replace(k, "-" + v)
            else:
                obj.replace(k, "-" + v + "-")
        return obj
    elif isinstance(obj, dict):
        return "-".join("%s-%s" % (sanitize(k), sanitize(v)) for k, v in sorted(obj.items()))
    else:
        cls = obj.__class__
        count = counters.get(cls, 0)
        counters[cls] = count + 1
        if count == 0:
            return cls.__name__
        else:
            return cls.__name__ + "-%s" %  count

def get_leafs(type):
    for sc in type.__subclasses__():
        if not inspect.isabstract(sc):
            yield sc
        for ssc in get_leafs(sc):
            yield ssc

def _fixup(var, cls, axis):
    var.cls = cls
    var.axis = axis
    return var

def variants(cls, *args, **kwargs):
    axis = kwargs.pop("axis", None)
    return tuple(_fixup(a, t, axis) for t in get_leafs(cls) for a in t.variants(*args, **kwargs))

def _instantiate(o):
    if isinstance(o, variant):
        return o.instantiate()
    elif isinstance(o, tuple):
        return tuple(_instantiate(i) for i in o)
    elif isinstance(o, list):
        return [_instantiate(i) for i in o]
    elif isinstance(o, dict):
        return {_instantiate(k): _instantiate(v) for k, v in o.items()}
    else:
        return o

# XXX: maybe make this a metaclass?
class variant:

    def __init__(self, *args, **kwargs):
        for a in args:
            assert not hasattr(a, "__next__")
        self.args = args
        self.kwargs = kwargs
        self.name = self.kwargs.pop("name", "")

    def clone(self, name):
        dict(self.kwargs)
        result = variant(*self.args, name=name, **self.kwargs)
        result.cls = self.cls
        result.axis = self.axis
        return result

    def instantiate(self):
        result = self.cls(*_instantiate(self.args))

        name = self.cls.__name__
        if self.name:
            name += "-" + result.format(self.name)
        if self.axis:
            name += "-" + result.format(self.axis)

        result.name = Name(name)

        names = {}
        for c in result.children:
            assert c.name not in names, (result, c, names[c.name])
            names[c.name] = c

        return result

def _set_parent(c, parent):
    if isinstance(c, Node):
        assert c.parent is None, (c.parent, c)
        c.parent = parent
        parent.children.append(c)
    elif isinstance(c, (tuple, list)):
        for o in c:
            _set_parent(o, parent)
    elif isinstance(c, dict):
        for k, v in c.items():
            _set_parent(k, parent)
            _set_parent(v, parent)

class Name(str):

    @property
    def k8s(self):
        return self.replace(".", "-").lower()

class Node(ABC):

    parent: 'Test'
    children: Sequence['Test']
    name: str

    def __new__(cls, *args, **kwargs):
        result = ABC.__new__(cls)
        result.parent = None
        result.children = []
        for a in args:
            _set_parent(a, result)
        return result

    @classmethod
    def variants(cls):
        return variant()

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
    def depth(self):
        if self.parent is None:
            return 0
        else:
            return self.parent.depth + 1

    def format(self, st):
        return st.format(self=self)

    @functools.lru_cache()
    def matches(self, pattern):
        if fnmatch.fnmatch(self.path, "*%s*" % pattern):
            return True
        for c in self.children:
            if c.matches(pattern):
                return True
        return False

class Test(Node):
    pass

class Query:

    def __init__(self, url, expected=200, skip = None, xfail = None):
        self.url = url
        self.expected = expected
        self.skip = skip
        self.xfail = xfail
        self.parent = None
        self.result = None

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
        assert self.query.expected == self.status, (self.query.expected, self.status or self.error)

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

class BackendResponse:

    def __init__(self, resp):
        self.headers = resp.get("headers", {})

class BackendResult:

    def __init__(self, bres):
        self.name = bres["backend"]
        self.request = BackendRequest(bres["request"]) if "request" in bres else None
        self.response = BackendResponse(bres["response"]) if "response" in bres else None
