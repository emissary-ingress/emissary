from ambassador.utils import RichStatus
from typing import Any, ClassVar, Dict, List, Optional, Type, Union, TYPE_CHECKING

from ..config import Config

from .irbasemapping import IRBaseMapping
from .irbasemappinggroup import IRBaseMappingGroup
from .irhttpmappinggroup import IRHTTPMappingGroup
from .ircors import IRCORS

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
    method: Optional[str]
    service: str
    group_id: str
    route_weight: List[Union[str, int]]
    cors: IRCORS
    sni: bool

    AllowedKeys: ClassVar[Dict[str, bool]] = {
        "add_request_headers": True,
        "add_response_headers": True,
        "auto_host_rewrite": True,
        "case_sensitive": True,
        # "circuit_breaker": True,
        "cors": True,
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
        "method": True,
        "method_regex": True,
        "modules": True,
        # "outlier_detection": True,
        "path_redirect": True,
        # Do not include precedence.
        "prefix": True,
        "prefix_regex": True,
        "priority": True,
        "rate_limits": True,   # Only supported in v0, handled in setup
        "remove_response_headers": True,
        "resolver": True,
        # Do not include regex_headers.
        # Do not include rewrite.
        "service": True,
        "shadow": True,
        "connect_timeout_ms": True,
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

                 kind: str="IRMapping",
                 apiVersion: str="ambassador/v1",   # Not a typo! See below.
                 precedence: int=0,
                 rewrite: str="/",
                 **kwargs) -> None:
        # OK, this is a bit of a pain. We want to preserve the name and rkey and
        # such here, unlike most kinds of IRResource. So. Shallow copy the keys
        # we're going to allow from the incoming kwargs...

        new_args = {x: kwargs[x] for x in kwargs.keys() if x in IRHTTPMapping.AllowedKeys}

        # ...then set up the headers (since we need them to compute our group ID).
        hdrs = []

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

        if 'method' in kwargs:
            hdrs.append(Header(":method", kwargs['method'], kwargs.get('method_regex', False)))

        # ...and then init the superclass.
        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, location=location,
            kind=kind, name=name, apiVersion=apiVersion,
            headers=hdrs, precedence=precedence, rewrite=rewrite,
            **new_args
        )

        if ('circuit_breaker' in kwargs) or ('outlier_detection' in kwargs):
            self.post_error(RichStatus.fromError("circuit_breaker and outlier_detection are not supported"))

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

        # Likewise, labels is supported only in V1:
        if 'labels' in self:
            if self.apiVersion != 'ambassador/v1':
                self.post_error("labels supported only in ambassador/v1 Mapping resources")
                return False

        if 'rate_limits' in self:
            if self.apiVersion != 'ambassador/v0':
                self.post_error("rate_limits supported only in ambassador/v0 Mapping resources")
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

        return True

    @staticmethod
    def validate_load_balancer(load_balancer) -> bool:
        lb_policy = load_balancer.get('policy', None)

        is_valid = False
        if lb_policy == 'round_robin':
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

        return h.hexdigest()

    def _route_weight(self) -> List[Union[str, int]]:
        len_headers = 0

        for hdr in self.headers:
            len_headers += hdr.length()

        # For calculating the route weight, 'method' defaults to '*' (for historical reasons).

        weight = [ self.precedence, len(self.prefix), len_headers, self.prefix, self.get('method', 'GET') ]
        weight += [ hdr.key() for hdr in self.headers ]

        return weight
