from ambassador.utils import RichStatus
from ambassador.utils import ParsedService as Service

from typing import Any, ClassVar, Dict, List, Optional, Type, Union, TYPE_CHECKING

from ..config import Config

from .irbasemapping import IRBaseMapping
from .irbasemappinggroup import IRBaseMappingGroup
from .irhttpmappinggroup import IRHTTPMappingGroup
from .ircors import IRCORS
from .irretrypolicy import IRRetryPolicy

import json
import hashlib

if TYPE_CHECKING:
    from .ir import IR


# Kind of cheating here so that it's easy to json-serialize Headers.
class Header (dict):
    def __init__(self, name: str, value: Optional[str]=None, regex: Optional[bool]=False) -> None:
        super().__init__()
        self.name = name
        self.value = value
        self.regex = regex

    def __getattr__(self, key: str) -> Any:
        return self[key]

    def __setattr__(self, key: str, value: Any) -> None:
        self[key] = value

    def _get_value(self) -> str:
        return self.value or '*'

    def length(self) -> int:
        return len(self.name) + len(self._get_value()) + (1 if self.regex else 0)

    def key(self) -> str:
        return self.name + '-' + self._get_value()


class IRHTTPMapping (IRBaseMapping):
    prefix: str
    headers: List[Header]
    add_request_headers: Dict[str, str]
    method: Optional[str]
    service: str
    group_id: str
    route_weight: List[Union[str, int]]
    cors: IRCORS
    retry_policy: IRRetryPolicy
    sni: bool

    AllowedKeys: ClassVar[Dict[str, bool]] = {
        # Do not include add_request_headers
        "add_response_headers": True,
        "auto_host_rewrite": True,
        "case_sensitive": True,
        "circuit_breakers": True,
        "add_linkerd_headers": True,
        "cors": True,
        "retry_policy": True,
        "enable_ipv4": True,
        "enable_ipv6": True,
        "envoy_override": True,
        "grpc": True,
        # Do not include headers.
        "host": True,
        "host_redirect": True,
        "host_regex": True,
        "host_rewrite": True,
        "labels": True,       # Only supported in v1, handled in setup
        "load_balancer": True,
        "keepalive": True,
        "method": True,
        "method_regex": True,
        "modules": True,
        # "outlier_detection": True,
        "path_redirect": True,
        # Do not include precedence.
        "prefix": True,
        "prefix_regex": True,
        "prefix_exact": True,
        "priority": True,
        "rate_limits": True,   # Only supported in v0, handled in setup
        "remove_response_headers": True,
        "remove_request_headers": True,
        "resolver": True,
        # Do not include regex_headers.
        # Do not include rewrite.
        "service": True,
        "shadow": True,
        "connect_timeout_ms": True,
        "cluster_idle_timeout_ms": True,
        "timeout_ms": True,
        "idle_timeout_ms": True,
        "tls": True,
        "use_websocket": True,
        "weight": True,
        "bypass_auth": True,

        # Include the serialization, too.
        "serialization": True,
    }

    def __init__(self, ir: 'IR', aconf: Config,
                 rkey: str,      # REQUIRED
                 name: str,      # REQUIRED
                 location: str,  # REQUIRED
                 namespace: Optional[str] = None,
                 metadata_labels: Optional[Dict[str, str]] = None,
                 kind: str="IRHTTPMapping",
                 apiVersion: str="getambassador.io/v2",   # Not a typo! See below.
                 precedence: int=0,
                 rewrite: str="/",
                 cluster_tag: Optional[str]=None,
                 **kwargs) -> None:
        # OK, this is a bit of a pain. We want to preserve the name and rkey and
        # such here, unlike most kinds of IRResource. So. Shallow copy the keys
        # we're going to allow from the incoming kwargs...

        new_args = {x: kwargs[x] for x in kwargs.keys() if x in IRHTTPMapping.AllowedKeys}

        # ...then set up the headers (since we need them to compute our group ID).
        hdrs = []
        add_request_hdrs = kwargs.get('add_request_headers', {})

        if 'headers' in kwargs:
            for name, value in kwargs.get('headers', {}).items():
                if value is True:
                    hdrs.append(Header(name))
                else:
                    hdrs.append(Header(name, value))

        if 'regex_headers' in kwargs:
            for name, value in kwargs.get('regex_headers', {}).items():
                hdrs.append(Header(name, value, regex=True))

        if 'host' in kwargs:
            hdrs.append(Header(":authority", kwargs['host'], kwargs.get('host_regex', False)))
            self.tls_context = self.match_tls_context(kwargs['host'], ir)

        if 'service' in kwargs:
            svc = Service(ir.logger, kwargs['service'])

            if 'add_linkerd_headers' in kwargs:
                if kwargs['add_linkerd_headers'] is True:
                    add_request_hdrs['l5d-dst-override'] = svc.hostname_port
            else:
                if 'add_linkerd_headers' in ir.ambassador_module and ir.ambassador_module.add_linkerd_headers is True:
                    add_request_hdrs['l5d-dst-override'] = svc.hostname_port

        if 'method' in kwargs:
            hdrs.append(Header(":method", kwargs['method'], kwargs.get('method_regex', False)))

        # XXX BRUTAL HACK HERE:
        # If we _don't_ have an origination context, but our IR has an agent_origination_ctx,
        # force TLS origination because it's the agent. I know, I know. It's a hack.
        if ('tls' not in new_args) and ir.agent_origination_ctx:
            ir.logger.info(f"Mapping {name}: Agent forcing origination TLS context to {ir.agent_origination_ctx.name}")
            new_args['tls'] = ir.agent_origination_ctx.name

        # ...and then init the superclass.
        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, location=location,
            kind=kind, name=name, namespace=namespace, metadata_labels=metadata_labels,
            apiVersion=apiVersion, headers=hdrs, add_request_headers=add_request_hdrs,
            precedence=precedence, rewrite=rewrite, cluster_tag=cluster_tag,
            **new_args
        )

        if 'outlier_detection' in kwargs:
            self.post_error(RichStatus.fromError("outlier_detection is not supported"))

    @staticmethod
    def group_class() -> Type[IRBaseMappingGroup]:
        return IRHTTPMappingGroup

    def setup(self, ir: 'IR', aconf: Config) -> bool:
        if not super().setup(ir, aconf):
            return False

        # If we have CORS stuff, normalize it.
        if 'cors' in self:
            self.cors = IRCORS(ir=ir, aconf=aconf, location=self.location, **self.cors)

            if self.cors:
                self.cors.referenced_by(self)
            else:
                return False

        # If we have RETRY_POLICY stuff, normalize it.
        if 'retry_policy' in self:
            self.retry_policy = IRRetryPolicy(ir=ir, aconf=aconf, location=self.location, **self.retry_policy)

            if self.retry_policy:
                self.retry_policy.referenced_by(self)
            else:
                return False

        # Likewise, labels is supported only in V1+:
        if 'labels' in self:
            if self.apiVersion == 'getambassador.io/v0':
                self.post_error("labels not supported in getambassador.io/v0 Mapping resources")
                return False

        if 'rate_limits' in self:
            if self.apiVersion != 'getambassador.io/v0':
                self.post_error("rate_limits supported only in getambassador.io/v0 Mapping resources")
                return False

            # Let's turn this into a set of labels instead.
            labels = []
            rlcount = 0

            for rate_limit in self.pop('rate_limits', []):
                rlcount += 1

                # Since this is a V0 Mapping, prepend the static default stuff that we were implicitly
                # forcing back in the pre-0.50 days.

                label: List[Any] = [
                    'source_cluster',
                    'destination_cluster',
                    'remote_address'
                ]

                # Next up: old rate_limit "descriptor" becomes label "generic_key".
                rate_limit_descriptor = rate_limit.get('descriptor', None)

                if rate_limit_descriptor:
                    label.append({ 'generic_key': rate_limit_descriptor })

                # Header names get turned into omit-if-not-present header dictionaries.
                rate_limit_headers = rate_limit.get('headers', [])

                for rate_limit_header in rate_limit_headers:
                    label.append({
                        rate_limit_header: {
                            'header': rate_limit_header,
                            'omit_if_not_present': True
                        }
                    })

                labels.append({
                    'v0_ratelimit_%02d' % rlcount: label
                })

            if labels:
                domain = 'ambassador' if not ir.ratelimit else ir.ratelimit.domain
                self['labels'] = { domain: labels }

        if self.get('load_balancer', None) is not None:
            if not self.validate_load_balancer(self['load_balancer']):
                self.post_error("Invalid load_balancer specified: {}, invalidating mapping".format(self['load_balancer']))
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

    @staticmethod
    def validate_load_balancer(load_balancer) -> bool:
        lb_policy = load_balancer.get('policy', None)

        is_valid = False
        if lb_policy in ['round_robin', 'least_request']:
            if len(load_balancer) == 1:
                is_valid = True
        elif lb_policy in ['ring_hash', 'maglev']:
            if len(load_balancer) == 2:
                if 'cookie' in load_balancer:
                    cookie = load_balancer.get('cookie')
                    if 'name' in cookie:
                        is_valid = True
                elif 'header' in load_balancer:
                    is_valid = True
                elif 'source_ip' in load_balancer:
                    is_valid = True

        return is_valid

    def _group_id(self) -> str:
        # Yes, we're using a cryptographic hash here. Cope. [ :) ]

        h = hashlib.new('sha1')

        # This is an HTTP mapping.
        h.update('HTTP-'.encode('utf-8'))

        # method first, but of course method might be None. For calculating the
        # group_id, 'method' defaults to 'GET' (for historical reasons).

        method = self.get('method') or 'GET'
        h.update(method.encode('utf-8'))
        h.update(self.prefix.encode('utf-8'))

        for hdr in self.headers:
            h.update(hdr.name.encode('utf-8'))

            if hdr.value is not None:
                h.update(hdr.value.encode('utf-8'))

        if self.precedence != 0:
            h.update(str(self.precedence).encode('utf-8'))

        return h.hexdigest()

    def _route_weight(self) -> List[Union[str, int]]:
        len_headers = 0

        for hdr in self.headers:
            len_headers += hdr.length()

        # For calculating the route weight, 'method' defaults to '*' (for historical reasons).

        weight = [ self.precedence, len(self.prefix), len_headers, self.prefix, self.get('method', 'GET') ]
        weight += [ hdr.key() for hdr in self.headers ]

        return weight

    def summarize_errors(self) -> str:
        errors = self.ir.aconf.errors.get(self.rkey, [])
        errstr = "(no errors)"

        if errors:
            errstr = errors[0].get('error') or 'unknown error?'

            if len(errors) > 1:
                errstr += " (and more)"

        return errstr

    def status(self) -> Dict[str, str]:
        if not self.is_active():
            return { 'state': 'Inactive', 'reason': self.summarize_errors() }
        else:
            return { 'state': 'Running' }
