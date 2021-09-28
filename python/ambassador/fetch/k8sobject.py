from __future__ import annotations
from typing import Any, Dict, Iterator, Optional

import collections.abc
import dataclasses
import enum

from ..config import Config


@dataclasses.dataclass(frozen=True)
class KubernetesGVK:
    """
    Represents a Kubernetes resource type (API group, version and kind).
    """

    api_version: str
    kind: str

    @property
    def api_group(self) -> Optional[str]:
        # These are backward-indexed to support apiVersion: v1, which has a
        # version but no group.
        try:
            return self.api_version.split('/', 1)[-2]
        except IndexError:
            return None

    @property
    def version(self) -> str:
        return self.api_version.split('/', 1)[-1]

    @property
    def domain(self) -> str:
        if self.api_group:
            return f'{self.kind.lower()}.{self.api_group}'
        else:
            return self.kind.lower()

    @classmethod
    def for_ambassador(cls, kind: str, version: str = 'v2') -> KubernetesGVK:
        if 'alpha' in version:
            return cls(f'x.getambassador.io/{version}', kind)
        else:
            return cls(f'getambassador.io/{version}', kind)

    @classmethod
    def for_knative_networking(cls, kind: str) -> KubernetesGVK:
        return cls('networking.internal.knative.dev/v1alpha1', kind)


@enum.unique
class KubernetesObjectScope (enum.Enum):
    CLUSTER = enum.auto()
    NAMESPACE = enum.auto()


@dataclasses.dataclass(frozen=True)
class KubernetesObjectKey:
    """
    Represents a single Kubernetes resource by kind and name.
    """

    gvk: KubernetesGVK
    namespace: Optional[str]
    name: str

    @property
    def kind(self) -> str:
        return self.gvk.kind

    @property
    def scope(self) -> KubernetesObjectScope:
        return KubernetesObjectScope.CLUSTER if self.namespace is None else KubernetesObjectScope.NAMESPACE

    @classmethod
    def from_object_reference(cls, ref: Dict[str, Any]) -> KubernetesObjectKey:
        return cls(KubernetesGVK('v1', ref['kind']), ref.get('namespace'), ref['name'])


class KubernetesObject (collections.abc.Mapping):
    """
    Represents a raw object from Kubernetes.
    """

    def __init__(self, delegate: Dict[str, Any]) -> None:
        self.delegate = delegate

        try:
            self.gvk
            self.name
        except KeyError:
            raise ValueError('delegate is not a valid Kubernetes object')

    def __getitem__(self, key: str) -> Any:
        return self.delegate[key]

    def __iter__(self) -> Iterator[str]:
        return iter(self.delegate)

    def __len__(self) -> int:
        return len(self.delegate)

    @property
    def gvk(self) -> KubernetesGVK:
        return KubernetesGVK(self['apiVersion'], self['kind'])

    @property
    def kind(self) -> str:
        return self.gvk.kind

    @property
    def metadata(self) -> Dict[str, Any]:
        return self['metadata']

    @property
    def namespace(self) -> str:
        val = self.metadata.get('namespace')
        if val == '_automatic_':
            val = Config.ambassador_namespace
        elif val is None:
            raise AttributeError(f'{self.__class__.__name__} {self.gvk.domain} {self.name} has no namespace (it is cluster-scoped)')

        return val

    @property
    def name(self) -> str:
        return self.metadata['name']

    @property
    def key(self) -> KubernetesObjectKey:
        try:
            namespace: Optional[str] = self.namespace
        except AttributeError:
            namespace = None

        return KubernetesObjectKey(self.gvk, namespace, self.name)

    @property
    def scope(self) -> KubernetesObjectScope:
        return self.key.scope

    @property
    def generation(self) -> int:
        return self.metadata.get('generation', 1)

    @property
    def annotations(self) -> Dict[str, str]:
        return self.metadata.get('annotations', {})

    @property
    def ambassador_id(self) -> str:
        return self.annotations.get('getambassador.io/ambassador-id', 'default')

    @property
    def labels(self) -> Dict[str, str]:
        return self.metadata.get('labels', {})

    @property
    def spec(self) -> Dict[str, Any]:
        return self.get('spec', {})

    @property
    def status(self) -> Dict[str, Any]:
        return self.get('status', {})
