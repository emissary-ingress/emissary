from typing import Any, ClassVar, Dict, List, Optional, Tuple, Union, TYPE_CHECKING

from ..config import Config

from .irresource import IRResource

if TYPE_CHECKING:
    from .ir import IR


class IRBaseMapping (IRResource):
    group_id: str
    host: str
    route_weight: List[Union[str, int]]
    sni: bool
    cached_status: Optional[Dict[str, str]]
    status_update: Optional[Dict[str, str]]

    def __init__(self, ir: 'IR', aconf: Config,
                 rkey: str,      # REQUIRED
                 name: str,      # REQUIRED
                 location: str,  # REQUIRED
                 kind: str,      # REQUIRED
                 namespace: Optional[str] = None,
                 metadata_labels: Optional[Dict[str, str]] = None,
                 apiVersion: str="getambassador.io/v2",
                 precedence: int=0,
                 cluster_tag: Optional[str]=None,
                 **kwargs) -> None:
        # Default status.
        self.cached_status = None
        self.status_update = None

        # Init the superclass...
        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, location=location,
            kind=kind, name=name, namespace=namespace, metadata_labels=metadata_labels,
            apiVersion=apiVersion, precedence=precedence, cluster_tag=cluster_tag,
            **kwargs
        )

    def setup(self, ir: 'IR', aconf: Config) -> bool:
        # We assume that any subclass madness is managed already, so we can compute the group ID...
        self.group_id = self._group_id()

        # ...and the route weight.
        self.route_weight = self._route_weight()

        # We can also default the resolver, and scream if it doesn't match a resolver we
        # know about.
        if not self.get('resolver'):
            self.resolver = self.ir.ambassador_module.get('resolver', 'kubernetes-service')

        resolver = self.ir.get_resolver(self.resolver)

        if not resolver:
            self.post_error(f'resolver {self.resolver} is unknown!')
            return False

        self.ir.logger.debug("%s: GID %s route_weight %s, resolver %s" %
                             (self, self.group_id, self.route_weight, resolver))

        # And, of course, we can make sure that the resolver thinks that this Mapping is OK.
        if not resolver.valid_mapping(ir, self):
            # If there's trouble, the resolver should've already posted about it.
            return False

        return True

    def get_label(self, key: str) -> Optional[str]:
        labels = self.get('metadata_labels') or {}
        return labels.get(key) or None

    def status(self) -> Optional[str]:
        """
        Return the new status we should have. Subclasses would typically override
        this.

        :return: new status (may be None)
        """
        return None

    def check_status(self) -> None:
        crd_name = self.get_label('ambassador_crd')

        if not crd_name:
            return

        # OK, we're supposed to be a CRD. What status do we want, and
        # what do we have?

        wanted = self.status()

        if wanted != self.cached_status:
            self.ir.k8s_status_updates[crd_name] = ('Mapping', self.namespace, wanted)

    def _group_id(self) -> str:
        """ Compute the group ID for this Mapping. Must be defined by subclasses. """
        raise NotImplementedError("%s._group_id is not implemented?" %  self.__class__.__name__)

    def _route_weight(self) -> List[Union[str, int]]:
        """ Compute the route weight for this Mapping. Must be defined by subclasses. """
        raise NotImplementedError("%s._route_weight is not implemented?" %  self.__class__.__name__)

    def match_tls_context(self, host: str, ir: 'IR'):
        for context in ir.get_tls_contexts():
            hosts = context.get('hosts') or []

            for context_host in hosts:
                if context_host == host:
                    ir.logger.debug("Matched host {} with TLSContext {}".format(host, context.get('name')))
                    self.sni = True
                    return context

        return None
