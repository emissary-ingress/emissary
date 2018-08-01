from abc import ABC, abstractmethod
from itertools import chain, product
from typing import Any, Iterable, Mapping, Optional, Sequence, Type

import copy, fnmatch, functools, inspect, json, os, pprint, sys

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
        result.name = self.cls.__name__
        if self.name:
            result.name += "-" + result.format(self.name)
        if self.axis:
            result.name += "-" + result.format(self.axis)

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
            return self.name
        else:
            return self.parent.relpath(ancestor) + "." + self.name

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

import argparse, fnmatch

parser = argparse.ArgumentParser()
subparsers = parser.add_subparsers(dest="op", help="subcommands")
list_parser = subparsers.add_parser("list", help="list tests, configuration, and/or manifests")
setup_parser = subparsers.add_parser("setup", help="setup the current cluster for testing")
run_parser = subparsers.add_parser("run", help="run tests")

def common(p):
    p.add_argument("filter", nargs="?", default="*")

for v in subparsers.choices.values():
    common(v)

def cli(root, args = None):
    if args is None:
        args = sys.argv[1:]
    ns = parser.parse_args(args)
    vars = tuple(v.instantiate() for v in variants(root))
    globals()["do_%s" % ns.op](vars, ns)

def do_list(vars, args):
    for v in vars:
        for t in v.traversal:
            if isinstance(t, Test) and t.matches(args.filter):
                print("  "*t.depth + t.relpath(t.parent))

from parser import dump
def do_setup(vars, args):
    for v in vars:
        if v.matches(args.filter):
            print(dump(v.assemble(args.filter)), end="")

def do_run(vars, args):
    urls = []
    for t in vars:
        urls.extend(t.urls())
    with open("/tmp/urls.json", "w") as f:
        json.dump(urls, f)
    os.system("go run client.go -input /tmp/urls.json -output /tmp/results.json")
