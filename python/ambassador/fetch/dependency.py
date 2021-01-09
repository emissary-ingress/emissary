from typing import Any, Collection, Iterator, Mapping, MutableSet, Optional, Protocol, Sequence, Type, TypeVar

from collections import defaultdict
import copy
import dataclasses

from .k8sobject import KubernetesObject


class Dependency (Protocol):

    def watt_key(self) -> str: ...


class ServiceDependency (Dependency):

    ambassador_service: Optional[KubernetesObject]

    def watt_key(self) -> str:
        return 'service'


class SecretDependency (Dependency):

    def watt_key(self) -> str:
        return 'secret'


class IngressClassesDependency (Dependency):

    ingress_classes: MutableSet[str]

    def __init__(self):
        self.ingress_classes = set()

    def watt_key(self) -> str:
        return 'ingressclasses'


D = TypeVar('D', bound=Dependency)


class DependencyMapping (Protocol):

    def __contains__(self, key: Type[D]) -> bool: ...
    def __getitem__(self, key: Type[D]) -> D: ...


class DependencyInjector:

    wants: MutableSet[Type[Dependency]]
    provides: MutableSet[Type[Dependency]]
    deps: DependencyMapping

    def __init__(self, deps: DependencyMapping) -> None:
        self.wants = set()
        self.provides = set()
        self.deps = deps

    def want(self, cls: Type[D]) -> D:
        self.wants.add(cls)
        return self.deps[cls]

    def provide(self, cls: Type[D]) -> D:
        self.provides.add(cls)
        return self.deps[cls]


class DependencyGraph:

    @dataclasses.dataclass
    class Vertex:
        out: MutableSet[Any]
        in_count: int

    vertices: Mapping[Any, Vertex]

    def __init__(self) -> None:
        self.vertices = defaultdict(lambda: DependencyGraph.Vertex(out=set(), in_count=0))

    def connect(self, a: Any, b: Any) -> None:
        if b not in self.vertices[a].out:
            self.vertices[a].out.add(b)
            self.vertices[b].in_count += 1

    def traverse(self) -> Iterator[Any]:
        """
        Returns the items in this graph in topological order.
        """

        in_counts = {obj: vertex.in_count for obj, vertex in self.vertices.items()}

        # Find the roots of the graph.
        queue = [obj for obj, in_count in in_counts.items() if in_count == 0]
        while len(queue) > 0:
            cur = queue.pop(0)
            yield cur

            for obj in self.vertices[cur].out:
                in_counts[obj] -= 1
                if in_counts[obj] == 0:
                    queue.append(obj)
        else:
            raise ValueError('cyclic')


class DependencyManager:

    deps: DependencyMapping
    injectors: Mapping[Any, DependencyInjector]

    def __init__(self, deps: Collection[D]) -> None:
        self.deps = {dep.__class__: dep for dep in deps}
        self.injectors = defaultdict(lambda: DependencyInjector(self.deps))

    def for_instance(self, obj: Any) -> DependencyInjector:
        return self.injectors[obj]

    def sorted_watt_keys(self) -> Sequence[Dependency]:
        g = DependencyGraph()

        for obj, injector in self.injectors.items():
            for cls in injector.provides:
                g.connect(obj, cls)
            for cls in injector.wants:
                g.connect(cls, obj)

        return [self.deps[obj].watt_key() for obj in g.traverse() if obj in self.deps]
