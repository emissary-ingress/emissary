from typing import FrozenSet, List, Mapping, Set

import collections
import logging

from ..config import Config

from .dependency import DependencyInjector
from .k8sobject import KubernetesGVK, KubernetesObjectScope, KubernetesObjectKey, KubernetesObject
from .resource import ResourceManager


class KubernetesProcessor:
    """
    An abstract processor for Kubernetes objects that emit configuration
    resources.
    """

    def kinds(self) -> FrozenSet[KubernetesGVK]:
        # Override kinds to describe the types of resources this processor wants
        # to process.
        return frozenset()

    def _process(self, obj: KubernetesObject) -> None:
        # Override _process to handle a single resource. Note that the entry
        # point for _process is try_process; _process should not be called
        # directly.
        pass

    def _admit(self, obj: KubernetesObject) -> bool:
        # Override _admit to change the admission rules for a specific object
        # for this processor. The default rules allow admission unless the
        # configuration specifies a single namespace and the object namespace is
        # outside of that namespace.
        if obj.scope == KubernetesObjectScope.NAMESPACE:
            if Config.single_namespace and obj.namespace != Config.ambassador_namespace:
                # This should never happen in actual usage, since we shouldn't
                # be given things in the wrong namespace. However, in
                # development, this can happen a lot.
                return False

        return True

    def try_process(self, obj: KubernetesObject) -> bool:
        if obj.gvk not in self.kinds() or not self._admit(obj):
            return False

        self._process(obj)
        return True

    def finalize(self) -> None:
        # Override finalize to do processing at the end of the configuration
        # fetching.
        pass


class ManagedKubernetesProcessor(KubernetesProcessor):
    """
    An abstract processor that provides access to a resource manager.
    """

    manager: ResourceManager

    def __init__(self, manager: ResourceManager):
        self.manager = manager

    @property
    def aconf(self) -> Config:
        return self.manager.aconf

    @property
    def logger(self) -> logging.Logger:
        return self.manager.logger

    @property
    def deps(self) -> DependencyInjector:
        return self.manager.deps.for_instance(self)


class AggregateKubernetesProcessor(KubernetesProcessor):
    """
    This processor aggregates many other processors into a single convenient
    processor.
    """

    delegates: List[KubernetesProcessor]
    mapping: Mapping[KubernetesGVK, List[KubernetesProcessor]]

    def __init__(self, delegates: List[KubernetesProcessor]) -> None:
        self.delegates = delegates
        self.mapping = collections.defaultdict(list)

        for proc in self.delegates:
            for kind in proc.kinds():
                self.mapping[kind].append(proc)

    def kinds(self) -> FrozenSet[KubernetesGVK]:
        return frozenset(iter(self.mapping))

    def _process(self, obj: KubernetesObject) -> None:
        procs = self.mapping.get(obj.gvk, [])
        for proc in procs:
            proc.try_process(obj)

    def finalize(self) -> None:
        for proc in self.delegates:
            proc.finalize()


class DeduplicatingKubernetesProcessor(KubernetesProcessor):
    """
    This processor delegates work to another processor but prevents the same
    Kubernetes object from being processed multiple times.
    """

    delegate: KubernetesProcessor
    cache: Set[KubernetesObjectKey]

    def __init__(self, delegate: KubernetesProcessor) -> None:
        self.delegate = delegate
        self.cache = set()

    def kinds(self) -> FrozenSet[KubernetesGVK]:
        return self.delegate.kinds()

    def _process(self, obj: KubernetesObject) -> None:
        if obj.key in self.cache:
            return

        self.cache.add(obj.key)
        self.delegate.try_process(obj)

    def finalize(self) -> None:
        self.delegate.finalize()


class CountingKubernetesProcessor(KubernetesProcessor):
    """
    This processor increments a given configuration counter when it receives an
    object.
    """

    aconf: Config
    kind: KubernetesGVK
    key: str

    def __init__(self, aconf: Config, kind: KubernetesGVK, key: str) -> None:
        self.aconf = aconf
        self.kind = kind
        self.key = key

    def kinds(self) -> FrozenSet[KubernetesGVK]:
        return frozenset([self.kind])

    def _process(self, obj: KubernetesObject) -> None:
        self.aconf.incr_count(self.key)
