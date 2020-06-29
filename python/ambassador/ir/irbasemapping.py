import json

from typing import Any, ClassVar, Dict, List, Optional, Tuple, Union, TYPE_CHECKING

from ..config import Config

from .irresource import IRResource

if TYPE_CHECKING:
    from .ir import IR

def qualify_service_name(ir: 'IR', service: str, namespace: Optional[str]) -> str:
    fully_qualified = "." in service or "localhost" == service

    if namespace != ir.ambassador_namespace and namespace and not fully_qualified and not ir.ambassador_module.use_ambassador_namespace_for_service_resolution:
        # The target service name is not fully qualified.
        # We are most likely targeting a simple k8s svc with kube-dns resolution.
        # Make sure we actually resolve the service it's namespace, not the Ambassador process namespace.
        # 
        # Note well! The "unqualified" service here might contain a port number, so just appending 
        # the namespace won't end well. So start by checking for a port number...

        fields = service.split(":", 1)

        hostname = fields[0]
        port: Optional[str] = None

        if len(fields) > 1:
            port = fields[1]

        service = f"{hostname}.{namespace}"

        if port is not None:
            service += f":{port}"

        ir.logger.debug("KubernetesServiceResolver use_ambassador_namespace_for_service_resolution %s, fully qualified %s, upstream hostname %s" % (
            ir.ambassador_module.use_ambassador_namespace_for_service_resolution,
            fully_qualified,
            service
        ))
    
    return service

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

        if self.get('circuit_breakers', None) is None:
            self['circuit_breakers'] = ir.ambassador_module.circuit_breakers

        if self.get('circuit_breakers', None) is not None:
            if not self.validate_circuit_breakers(ir, self['circuit_breakers']):
                self.post_error("Invalid circuit_breakers specified: {}, invalidating mapping".format(self['circuit_breakers']))
                return False

        return True

    @staticmethod
    def validate_circuit_breakers(ir: 'IR', circuit_breakers) -> bool:
        if not isinstance(circuit_breakers, (list, tuple)):
            return False

        for circuit_breaker in circuit_breakers:
            if '_name' in circuit_breaker:
                # Already reconciled.
                ir.logger.debug(f'Breaker validation: good breaker {circuit_breaker["_name"]}')
                continue

            ir.logger.debug(f'Breaker validation: {json.dumps(circuit_breakers, indent=4, sort_keys=True)}')

            name_fields = [ 'cb' ]

            if 'priority' in circuit_breaker:
                prio = circuit_breaker.get('priority').lower()
                if prio not in ['default', 'high']:
                    return False

                name_fields.append(prio[0])
            else:
                name_fields.append('n')

            digit_fields = [ ( 'max_connections', 'c' ),
                             ( 'max_pending_requests', 'p' ),
                             ( 'max_requests', 'r' ),
                             ( 'max_retries', 't' ) ]

            for field, abbrev in digit_fields:
                if field in circuit_breaker:
                    try:
                        value = int(circuit_breaker[field])
                        name_fields.append(f'{abbrev}{value}')
                    except ValueError:
                        return False

            circuit_breaker['_name'] = ''.join(name_fields)
            ir.logger.debug(f'Breaker valid: {circuit_breaker["_name"]}')

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
