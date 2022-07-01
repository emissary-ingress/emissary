from enum import Enum, auto
from io import StringIO
from typing import Any, Callable, Mapping, Sequence, Type
from yaml import (
    compose,
    compose_all,
    dump_all,
    MappingNode,
    SequenceNode,
    ScalarNode,
    Node,
    add_representer,
)


class ViewMode(Enum):
    PYTHON = auto()
    STRING = auto()
    NODE = auto()


class Tag(Enum):
    SEQUENCE = compose("[]").tag
    MAPPING = compose("{}").tag
    STRING = compose("hello").tag
    INT = compose("3").tag
    FLOAT = compose("3.14159265359").tag
    BOOL = compose("true").tag
    NULL = compose("null").tag


class View:
    def __init__(self, node: Node, mode: ViewMode) -> None:
        self.node = node
        self.mode = mode

    @property
    def tag(self):
        return Tag(self.node.tag)

    def view(self, obj):
        return view(obj, self.mode)

    def mode_ify(self):
        return self


class MappingView(View, Mapping):
    def get(self, key, default=None):
        for k, v in self.node.value:
            if k.value == key:
                return self.view(v)
        return default

    def __contains__(self, key):
        for k, v in self.node.value:
            if k.value == key:
                return True
        return False

    def __getitem__(self, key):
        for k, v in self.node.value:
            if k.value == key:
                return self.view(v)
        raise KeyError(key)

    def __setitem__(self, key, value):
        for idx, (k, v) in enumerate(self.node.value):
            if k.value == key:
                self.node.value[idx] = (node(key), node(value))
                break
        else:
            self.node.value.append((node(key), node(value)))

    def update(self, other):
        for k, v in other.items():
            self[k] = v

    def merge(self, other):
        self.node.value.extend(other.node.value)

    def keys(self):
        return set(k.value for k, v in self.node.value)

    def items(self):
        for k, v in self.node.value:
            yield (self.view(k), self.view(v))

    def __iter__(self):
        for k, v in self.node.value:
            yield self.view(k)

    def __len__(self):
        return len(self.node.value)

    def __repr__(self):
        return "{%s}" % ", ".join(
            "%r: %r" % (view(k, ViewMode.PYTHON), view(v, ViewMode.PYTHON))
            for k, v in self.node.value
        )


class SequenceView(View, Sequence):
    def __getitem__(self, idx):
        return view(self.node.value[idx], self.mode)

    def __setitem__(self, idx, value):
        self.node.value[idx] = node(value)

    def append(self, value):
        self.node.value.append(node(value))

    def __len__(self):
        return len(self.node.value)

    def __iter__(self):
        for i in self.node.value:
            yield self.view(i)

    def extend(self, items):
        for i in items:
            self.append(i)

    def merge(self, other):
        self.node.value.extend(other.node.value)

    def __repr__(self):
        return repr([v for v in self])


PYJECTIONS = {
    Tag.INT: lambda x: int(x),
    Tag.FLOAT: lambda x: float(x),
    Tag.STRING: lambda x: x,
    Tag.BOOL: lambda x: x.lower() in ("y", "yes", "true", "on"),
    Tag.NULL: lambda x: None,
}


class ScalarView(View):
    def mode_ify(self):
        if self.mode == ViewMode.PYTHON:
            return PYJECTIONS[Tag(self.tag)](self.node.value)
        elif self.mode == ViewMode.STRING:
            return self.node.value
        else:
            return self

    def __repr__(self):
        return self.node.value


VIEWS: Mapping[Type[Node], Type[View]] = {
    MappingNode: MappingView,
    SequenceNode: SequenceView,
    ScalarNode: ScalarView,
}


def view(value: Any, mode: ViewMode) -> Any:
    nd = node(value)
    return VIEWS[type(nd)](nd, mode).mode_ify()


COERCIONS: Mapping[Type, Callable[[Any], Node]] = {
    MappingNode: lambda n: n,
    SequenceNode: lambda n: n,
    ScalarNode: lambda n: n,
    MappingView: lambda v: v.node,
    SequenceView: lambda v: v.node,
    ScalarView: lambda v: v.node,
    list: lambda l: SequenceNode(Tag.SEQUENCE.value, [node(i) for i in l]),
    tuple: lambda t: SequenceNode(Tag.SEQUENCE.value, [node(i) for i in t]),
    str: lambda s: ScalarNode(Tag.STRING.value, str(s)),
    bool: lambda b: ScalarNode(Tag.BOOL.value, str(b)),
    int: lambda i: ScalarNode(Tag.INT.value, str(i)),
    float: lambda f: ScalarNode(Tag.FLOAT.value, str(f)),
    dict: lambda d: MappingNode(Tag.MAPPING.value, [(node(k), node(v)) for k, v in d.items()]),
}


def node(value: Any) -> Node:
    return COERCIONS[type(value)](value)


def load(name: str, value: Any, *allowed: Tag) -> SequenceView:
    if isinstance(value, str):
        value = StringIO(value)
        value.name = name
    result = view(SequenceNode(Tag.SEQUENCE.value, list(compose_all(value))), ViewMode.PYTHON)
    for r in view(result, ViewMode.NODE):
        if r.tag not in allowed:
            raise ValueError(
                "expecting %s, got %s" % (", ".join(t.name for t in allowed), r.node.tag)
            )
    return result


def dump(value: SequenceView):
    st = dump_all(value, default_flow_style=False)
    if not st.startswith("---"):
        st = "---\n" + st
    return st


def view_representer(dumper, data):
    return data.node


add_representer(SequenceView, view_representer)
add_representer(MappingView, view_representer)
add_representer(ScalarView, view_representer)
