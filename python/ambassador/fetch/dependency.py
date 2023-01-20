import dataclasses
from collections import defaultdict
from typing import (
    Any,
    Collection,
    Dict,
    Iterator,
    Mapping,
    MutableSet,
    Optional,
    Protocol,
    Sequence,
    Type,
    TypeVar,
)

from .k8sobject import KubernetesObjectKey, KubernetesObject


class Dependency(Protocol):
    """
    Dependencies link information provided by processors of a given Watt
    invocation to other processors that need the processed result. This results
    in an ordering of keys so that processors can be dependent on each other
    without direct knowledge of where data is coming from.
    """

    def watt_key(self) -> str:
        ...


class ServiceDependency(Dependency):
    """
    A dependency that exposes information about the Kubernetes service for
    Ambassador itself.
    """

    ambassador_service: Optional[KubernetesObject]
    discovered_services: Dict[KubernetesObjectKey, KubernetesObject]

    def __init__(self) -> None:
        self.ambassador_service = None
        self.discovered_services = {}

    def watt_key(self) -> str:
        return "service"


class SecretDependency(Dependency):
    """
    A dependency that is satisfied once secret information has been mapped and
    emitted.
    """

    def watt_key(self) -> str:
        return "secret"


class IngressClassesDependency(Dependency):
    """
    A dependency that provides the list of ingress classes that are valid (i.e.,
    have the proper controller) for this cluster.
    """

    ingress_classes: MutableSet[str]

    def __init__(self):
        self.ingress_classes = set()

    def watt_key(self) -> str:
        return "ingressclasses"


D = TypeVar("D", bound=Dependency)


class DependencyMapping(Protocol):
    def __contains__(self, key: Type[D]) -> bool:
        ...

    def __getitem__(self, key: Type[D]) -> D:
        ...


class DependencyInjector:
    """
    Each processor instance is provided with a dependency injector that allows
    it to declare what dependencies it provides as part of its processing and
    what dependencies it wants to do its processing.

    Note that dependencies need not be fulfilled; for example, nothing may
    provide information about the Ambassador service or the list of valid
    ingress classes. Processors should be prepared to deal with the absence of
    valid data when they run.
    """

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
    """
    Once dependency relationships are known, this class provides the ability to
    link them holistically and traverse them in topological order. It is most
    useful in the context of the sorted_watt_keys() method of the
    DependencyManager.
    """

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

        if len(self.vertices) == 0:
            return

        # This method implements Kahn's algorithm. See
        # https://en.wikipedia.org/wiki/Topological_sorting#Kahn's_algorithm for
        # more information.

        # Create a copy of the counts of each inbound edge so we can mutate
        # them.
        in_counts = {obj: vertex.in_count for obj, vertex in self.vertices.items()}

        # Find the roots of the graph.
        queue = [obj for obj, in_count in in_counts.items() if in_count == 0]

        # No roots of a graph with at least one vertex indicates a cycle.
        if len(queue) == 0:
            raise ValueError("cyclic")

        while len(queue) > 0:
            cur = queue.pop(0)
            yield cur

            for obj in self.vertices[cur].out:
                in_counts[obj] -= 1
                if in_counts[obj] == 0:
                    queue.append(obj)

        assert sum(in_counts.values()) == 0, "Traversal did not reach every vertex exactly once"


class DependencyManager:
    """
    A manager that provides access to a set of dependencies for arbitrary object
    instances and the ability to compute a sorted list of Watt keys that
    represent the processing order for the dependencies.
    """

    deps: DependencyMapping
    injectors: Mapping[Any, DependencyInjector]

    def __init__(self, deps: Collection[D]) -> None:
        self.deps = {dep.__class__: dep for dep in deps}
        self.injectors = defaultdict(lambda: DependencyInjector(self.deps))

    def for_instance(self, obj: Any) -> DependencyInjector:
        return self.injectors[obj]

    def sorted_watt_keys(self) -> Sequence[str]:
        g = DependencyGraph()

        for obj, injector in self.injectors.items():
            for cls in injector.provides:
                g.connect(obj, cls)
            for cls in injector.wants:
                g.connect(cls, obj)

        return [self.deps[obj].watt_key() for obj in g.traverse() if obj in self.deps]
