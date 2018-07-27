from abc import ABC, abstractmethod
from itertools import chain, product
from typing import Any, Iterable, Optional, Sequence, Type

import inspect, sys

from parser import load, dump, Tag, SequenceView

def get_type(type):
    for m in sys.modules.values():
        for k, v in m.__dict__.items():
            if inspect.isclass(v):
                if issubclass(v, type) and v != type and not inspect.isabstract(v):
                    yield v

def expand(c):
    if isinstance(c, choice):
        for alt in c.alternatives:
            for a in expand(alt):
                yield a
    elif isinstance(c, tuple) and c:
        for alt1 in expand(c[0]):
            for alt2 in expand(c[1:]):
                yield (alt1,) + alt2
    else:
        yield c


def set_parent(c, parent):
    if isinstance(c, Test):
        c.parent = parent
        parent.children.append(c)
    elif isinstance(c, tuple):
        for o in c:
            set_parent(o, parent)

def instantiate(factory, args):
    try:
        result = factory(*args)
    except Exception as e:
        raise Exception("error instantiating %r with args %s: %s" % (factory, ", ".join(repr(r) for r in args), e))

    result.parent = None
    result.children = []

    for a in args:
        set_parent(a, result)

    return result

flatten = chain.from_iterable

def variants(cls):
    return tuple(variant(*f) for f in flatten(expand(c) for t in get_type(cls) for c in t.variants()))

def _instantiate(v):
    if isinstance(v, variant):
        return instantiate(v.factory, _instantiate(v.args))
    elif isinstance(v, tuple):
        return tuple(_instantiate(o) for o in v)
    else:
        return v

class variant:

    def __init__(self, factory, *args):
        self.factory = factory
        self.args = args

    def instantiate(self):
        return _instantiate(self)

class choice:

    def __init__(self, alternatives):
        self.alternatives = tuple(alternatives)

    def __repr__(self):
        return "choice(%s)" % (", ".join(repr(a) for a in self.alternatives))

class Test(ABC):

    parent: 'Test'
    children: Sequence['Test']

    def name(self) -> str:
        return self.__class__.__name__

    def path(self) -> str:
        if self.parent is None:
            return self.name()
        else:
            return self.parent.path() + "." + self.name()

    def list(self, level = 0):
        print("  "*level + self.path())
        for c in self.children:
            c.list(level = level + 1)

    @abstractmethod
    def yaml(self) -> str:
        pass

    def yaml_check(self, gen, *tags: Tag) -> Optional[SequenceView]:
        st = gen()
        if st is None: return None
        seq = load(self.name(), st)
        for o in seq:
            if o.tag not in tags:
                raise ValueError("test %s expecting %s, got %s" % (self.name(), ", ".join(t.name for t in tags),
                                                                   o.node.tag))
        return seq
