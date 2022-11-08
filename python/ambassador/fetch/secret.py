from typing import FrozenSet

from ..config import Config
from .dependency import SecretDependency
from .k8sobject import KubernetesGVK, KubernetesObject
from .k8sprocessor import ManagedKubernetesProcessor
from .resource import NormalizedResource, ResourceManager


class SecretProcessor(ManagedKubernetesProcessor):
    """
    A Kubernetes object processor that emits Ambassador secrets from Kubernetes secrets.
    """

    KNOWN_TYPES = [
        "Opaque",
        "kubernetes.io/tls",
        "istio.io/key-and-cert",
    ]

    KNOWN_DATA_KEYS = [
        "tls.crt",  # type="kubernetes.io/tls"
        "tls.key",  # type="kubernetes.io/tls"
        "user.key",  # type="Opaque", used for AES ACME
        "cert-chain.pem",  # type="istio.io/key-and-cert"
        "key.pem",  # type="istio.io/key-and-cert"
        "root-cert.pem",  # type="istio.io/key-and-cert"
        "crl.pem",  # type="Opaque", used for TLS CRL
    ]

    def __init__(self, manager: ResourceManager) -> None:
        super().__init__(manager)

        self.deps.provide(SecretDependency)

    def kinds(self) -> FrozenSet[KubernetesGVK]:
        return frozenset([KubernetesGVK("v1", "Secret")])

    def _admit(self, obj: KubernetesObject) -> bool:
        if not Config.certs_single_namespace:
            return True

        return super()._admit(obj)

    def _process(self, obj: KubernetesObject) -> None:
        # self.logger.debug("processing K8s Secret %s", dump_json(dict(obj), pretty=True))

        secret_type = obj.get("type")
        if secret_type not in self.KNOWN_TYPES:
            self.logger.debug("ignoring K8s Secret with unknown type %s" % secret_type)
            return

        data = obj.get("data")
        if not data:
            self.logger.debug("ignoring K8s Secret with no data")
            return

        if not any(key in data for key in self.KNOWN_DATA_KEYS):
            # Uh. WTFO?
            self.logger.debug(f"ignoring K8s Secret {obj.name}.{obj.namespace} with no keys")
            return

        spec = {
            "ambassador_id": Config.ambassador_id,
            "secret_type": secret_type,
        }

        for key, value in data.items():
            spec[key.replace(".", "_")] = value

        self.manager.emit(
            NormalizedResource.from_data(
                "Secret",
                obj.name,
                namespace=obj.namespace,
                labels=obj.labels,
                spec=spec,
                errors=obj.get("errors"),  # Make sure we preserve errors here!
            )
        )
