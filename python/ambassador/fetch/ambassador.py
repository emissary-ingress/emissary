from typing import FrozenSet

import itertools

from .k8sobject import KubernetesGVK, KubernetesObject
from .k8sprocessor import ManagedKubernetesProcessor
from .resource import NormalizedResource


class AmbassadorProcessor (ManagedKubernetesProcessor):
    """
    A Kubernetes object processor that emits direct IR from an Ambassador CRD.
    """

    def kinds(self) -> FrozenSet[KubernetesGVK]:
        kinds = [
            'AuthService',
            'ConsulResolver',
            'Host',
            'KubernetesEndpointResolver',
            'KubernetesServiceResolver',
            'Listener',
            'LogService',
            'Mapping',
            'Module',
            'RateLimitService',
            'DevPortal',
            'TCPMapping',
            'TLSContext',
            'TracingService',
        ]

        return frozenset([
            KubernetesGVK.for_ambassador(kind, version=version) for (kind, version) in itertools.product(kinds, ['v1', 'v2', 'v3alpha1'])
        ])

    def _process(self, obj: KubernetesObject) -> None:
        self.manager.emit(NormalizedResource.from_kubernetes_object(obj))
