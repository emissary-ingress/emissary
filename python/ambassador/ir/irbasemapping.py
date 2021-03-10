import json

from typing import Any, ClassVar, Dict, List, Optional, Tuple, Union, TYPE_CHECKING

from urllib.parse import scheme_chars, urlparse

from ..config import Config
from ..utils import dump_json

from .irresource import IRResource

if TYPE_CHECKING:
    from .ir import IR

def would_confuse_urlparse(url: str) -> bool:
    """Returns whether an URL-ish string would be interpretted by urlparse()
    differently than we want, by parsing it as a non-URL URI ("scheme:path")
    instead of as a URL ("[scheme:]//authority[:port]/path").  We don't want to
    interpret "myhost:8080" as "ParseResult(scheme='myhost', path='8080')"!

    """
    if url.find(':') > 0 and url.lstrip(scheme_chars).startswith("://"):
        # has a scheme
        return False
    if url.startswith("//"):
        # does not have a scheme, but has the "//" URL authority marker
        return False
    return True

def normalize_service_name(ir: 'IR', in_service: str, mapping_namespace: Optional[str], resolver_kind: str, rkey: Optional[str]=None) -> str:
    try:
        parsed = urlparse(f"//{in_service}" if would_confuse_urlparse(in_service) else in_service)

        hostname = parsed.hostname
        if not hostname:
            raise ValueError("No hostname")
        scheme = parsed.scheme
        port = parsed.port
    except ValueError as e:
        # This could happen with mismatched [] in a scheme://[IPv6], or with a port that can't
        # cast to int, or a port outside [0,2^16), or...
        #
        # The best we can do here is probably just to log the error, return the original string
        # and hope for the best. I guess.

        errstr = f"Malformed service {repr(in_service)}: {e}"
        if rkey:
            errstr = f"{rkey}: {errstr}"
        ir.post_error(errstr)

        return in_service

    # Consul Resolvers don't allow service names to include subdomains, but
    # Kubernetes Resolvers _require_ subdomains to correctly handle namespaces.
    want_qualified = not ir.ambassador_module.use_ambassador_namespace_for_service_resolution and resolver_kind.startswith('Kubernetes')

    is_qualified = "." in hostname or "localhost" == hostname

    if mapping_namespace and mapping_namespace != ir.ambassador_namespace and want_qualified and not is_qualified:
        hostname += "."+mapping_namespace

    out_service = hostname
    if scheme:
        out_service = f"{scheme}://{out_service}"
    if port:
        out_service += f":{port}"

    ir.logger.debug("%s use_ambassador_namespace_for_service_resolution %s, fully qualified %s, upstream hostname %s" % (
        resolver_kind,
        ir.ambassador_module.use_ambassador_namespace_for_service_resolution,
        is_qualified,
        out_service
    ))
    
    return out_service

class IRBaseMapping (IRResource):
    group_id: str
    host: str
    route_weight: List[Union[str, int]]
    sni: bool
    cached_status: Optional[Dict[str, str]]
    status_update: Optional[Dict[str, str]]
    cluster_key: Optional[str]

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
        # Default status...
        self.cached_status = None
        self.status_update = None

        # Start by assuming that we don't know the cluster key for this Mapping.
        self.cluster_key = None

        # Init the superclass...
        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, location=location,
            kind=kind, name=name, namespace=namespace, metadata_labels=metadata_labels,
            apiVersion=apiVersion, precedence=precedence, cluster_tag=cluster_tag,
            **kwargs
        )

    @classmethod
    def make_cache_key(cls, kind: str, name: str, namespace: str, version: str="v2") -> str:
        return f"{kind}-{version}-{name}-{namespace}"

    def setup(self, ir: 'IR', aconf: Config) -> bool:
        # Set up our cache key. We're using this format so that it'll be easy
        # to generate it just from the Mapping's K8s metadata.
        self._cache_key = IRBaseMapping.make_cache_key(self.kind, self.name, self.namespace)

        # ...and start without a cluster key for this Mapping.
        self.cluster_key = None

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

            ir.logger.debug(f'Breaker validation: {dump_json(circuit_breakers, pretty=True)}')

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

    def status(self) -> Optional[Dict[str, Any]]:
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
