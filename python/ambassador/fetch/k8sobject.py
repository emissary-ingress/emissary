from __future__ import annotations
from typing import Any, Dict, Iterator, Optional

import collections.abc
import dataclasses

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
        return cls(f'getambassador.io/{version}', kind)

    @classmethod
    def for_knative_networking(cls, kind: str) -> KubernetesGVK:
        return cls('networking.internal.knative.dev/v1alpha1', kind)


@dataclasses.dataclass(frozen=True)
class KubernetesObjectKey:
    """
    Represents a single Kubernetes resource by kind and name.
    """

    gvk: KubernetesGVK
    namespace: Optional[str]
    name: str


class KubernetesObject(collections.abc.Mapping):
    """
    Represents a raw object from Kubernetes.
    """

    default_namespace: Optional[str]

    def __init__(self, delegate: Dict[str, Any], default_namespace: Optional[str] = None) -> None:
        self.delegate = delegate
        self.default_namespace = default_namespace

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
    def namespace(self) -> Optional[str]:
        val = self.metadata.get('namespace', self.default_namespace)
        if val == '_automatic_':
            val = Config.ambassador_namespace

        return val

    @property
    def name(self) -> str:
        return self.metadata['name']

    @property
    def key(self) -> KubernetesObjectKey:
        return KubernetesObjectKey(self.gvk, self.namespace, self.name)

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
