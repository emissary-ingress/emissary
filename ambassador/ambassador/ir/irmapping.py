from ambassador.utils import RichStatus
from typing import Any, ClassVar, Dict, List, Optional, Union, TYPE_CHECKING

from ..config import Config

from .irresource import IRResource
from .ircluster import IRCluster

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


class IRMapping (IRResource):
    prefix: str
    headers: List[Header]
    method: Optional[str]
    service: str
    group_id: str
    route_weight: List[Union[str, int]]

    AllowedKeys: ClassVar[Dict[str, bool]] = {
        "add_request_headers": True,
        "auto_host_rewrite": True,
        "case_sensitive": True,
        "circuit_breaker": True,
        "cors": True,
        "envoy_override": True,
        "grpc": True,
        # Do not include headers.
        "host": True,
        "host_redirect": True,
        "host_regex": True,
        "host_rewrite": True,
        "method": True,
        "method_regex": True,
        "modules": True,
        "outlier_detection": True,
        "path_redirect": True,
        # Do not include precedence.
        "prefix": True,
        "prefix_regex": True,
        "priority": True,
        "rate_limits": True,
        # Do not include regex_headers.
        # Do not include rewrite.
        "service": True,
        "shadow": True,
        "timeout_ms": True,
        "tls": True,
        "use_websocket": True,
        "weight": True,
    }

    def __init__(self, ir: 'IR', aconf: Config,
                 rkey: str,      # REQUIRED
                 name: str,      # REQUIRED
                 location: str,  # REQUIRED

                 kind: str="IRMapping",
                 apiVersion: str="ambassador/v0",   # Not a typo! See below.
                 precedence: int=0,
                 rewrite: str="/",
                 **kwargs) -> None:
        # OK, this is a bit of a pain. We want to preserve the name and rkey and
        # such here, unlike most kinds of IRResource. So. Shallow copy the keys
        # we're going to allow from the incoming kwargs...

        new_args = { x: kwargs[x] for x in kwargs.keys() if x in IRMapping.AllowedKeys }

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

        if 'method' in kwargs:
            hdrs.append(Header(":method", kwargs['method'], kwargs.get('method_regex', False)))

        # ...and then init the superclass.
        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, location=location,
            kind=kind, name=name, apiVersion=apiVersion,
            headers=hdrs, precedence=precedence, rewrite=rewrite,
            **new_args
        )

        # OK. After all that we can compute the group ID...
        self.group_id = self._group_id()

        # ...and the route weight.
        self.route_weight = self._route_weight()

    def _group_id(self) -> str:
        # Yes, we're using a cryptographic hash here. Cope. [ :) ]

        h = hashlib.new('sha1')

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

    def _route_weight(self):
        len_headers = 0

        for hdr in self.headers:
            len_headers += hdr.length()

        # For calculating the route weight, 'method' defaults to '*' (for historical reasons).

        weight = [ self.precedence, len(self.prefix), len_headers, self.prefix, self.get('method', 'GET') ]
        weight += [ hdr.key() for hdr in self.headers ]

        return tuple(weight)

    # def save_cors_element(self, cors_key, route_key, route):
    #     """If self.get('cors')[cors_key] exists, and
    #     - is a list, e.g. ["1","2","3"], then route[route_key] is set as "1, 2, 3"
    #     - is something else, then set route[route_key] as it is
    #
    #     :param cors_key: key to exist in self.get('cors'), i.e. Ambassador's config
    #     :param route_key: key to save to in envoy's cors configuration
    #     :param route: envoy's cors configuration
    #     """
    #     cors = self.get('cors')
    #     if cors.get(cors_key) is not None:
    #         if type(cors.get(cors_key)) is list:
    #             route[ route_key ] = ", ".join(cors.get(cors_key))
    #         else:
    #             route[ route_key ] = cors.get(cors_key)
    #
    # def generate_route_cors(self):
    #     """Generates envoy's cors configuration from ambassador's cors configuration
    #
    #     :return generated envoy cors configuration
    #     :rtype: dict
    #     """
    #
    #     cors = self.get('cors')
    #     if cors is None:
    #         return
    #
    #     route_cors = {'enabled': True}
    #     # cors['origins'] cannot be treated like other keys, because if it's a
    #     # list, then it remains as is, but if it's a string, then it's
    #     # converted to a list
    #     origins = cors.get('origins')
    #     if origins is not None:
    #         if type(origins) is list:
    #             route_cors[ 'allow_origin' ] = origins
    #         elif type(origins) is str:
    #             route_cors[ 'allow_origin' ] = origins.split(',')
    #         else:
    #             print("invalid cors configuration supplied - {}".format(origins))
    #             return
    #
    #     self.save_cors_element('max_age', 'max_age', route_cors)
    #     self.save_cors_element('credentials', 'allow_credentials', route_cors)
    #     self.save_cors_element('methods', 'allow_methods', route_cors)
    #     self.save_cors_element('headers', 'allow_headers', route_cors)
    #     self.save_cors_element('exposed_headers', 'expose_headers', route_cors)
    #     return route_cors
    #
    #
    # def new_route(self, svc, cluster_name) -> SourcedDict:
    #     route = SourcedDict(
    #         _source=self[ '_source' ],
    #         _group_id=self.group_id,
    #         _precedence=self.get('precedence', 0),
    #         prefix_rewrite=self.get('rewrite', '/')
    #     )
    #
    #     if self.get('prefix_regex', False):
    #         # if `prefix_regex` is true, then use the `prefix` attribute as the envoy's regex
    #         route[ 'regex' ] = self[ 'prefix' ]
    #     else:
    #         route[ 'prefix' ] = self[ 'prefix' ]
    #
    #     host_redirect = self.get('host_redirect', False)
    #     shadow = self.get('shadow', False)
    #
    #     if not host_redirect and not shadow:
    #         route[ 'clusters' ] = [ {"name": cluster_name,
    #                                  "weight": self.get("weight", None)} ]
    #     else:
    #         route.setdefault('clusters', [ ])
    #
    #         if host_redirect and not shadow:
    #             route[ 'host_redirect' ] = svc
    #             route.setdefault('clusters', [ ])
    #         elif shadow:
    #             # If both shadow and host_redirect are set, we let shadow win.
    #             #
    #             # XXX CODE DUPLICATION with config.py!!
    #             # We're going to need to support shadow weighting later, so use a dict here.
    #             route[ 'shadow' ] = {
    #                 'name': cluster_name
    #             }
    #
    #     if self.headers:
    #         route[ 'headers' ] = self.headers
    #
    #     add_request_headers = self.get('add_request_headers')
    #     if add_request_headers:
    #         route[ 'request_headers_to_add' ] = [ ]
    #         for key, value in add_request_headers.items():
    #             route[ 'request_headers_to_add' ].append({"key": key, "value": value})
    #
    #     envoy_cors = self.generate_route_cors()
    #     if envoy_cors:
    #         route[ 'cors' ] = envoy_cors
    #
    #     rate_limits = self.get('rate_limits')
    #
    #     if rate_limits:
    #         route[ 'rate_limits' ] = [ ]
    #         for rate_limit in rate_limits:
    #             rate_limits_actions = [
    #                 {'type': 'source_cluster'},
    #                 {'type': 'destination_cluster'},
    #                 {'type': 'remote_address'}
    #             ]
    #
    #             rate_limit_descriptor = rate_limit.get('descriptor', None)
    #
    #             if rate_limit_descriptor:
    #                 rate_limits_actions.append({'type': 'generic_key',
    #                                             'descriptor_value': rate_limit_descriptor})
    #
    #             rate_limit_headers = rate_limit.get('headers', [ ])
    #
    #             for rate_limit_header in rate_limit_headers:
    #                 rate_limits_actions.append({'type': 'request_headers',
    #                                             'header_name': rate_limit_header,
    #                                             'descriptor_key': rate_limit_header})
    #
    #             route[ 'rate_limits' ].append({'actions': rate_limits_actions})
    #
    #     # Even though we don't use it for generating the Envoy config, go ahead
    #     # and make sure that any ':method' header match gets saved under the
    #     # route's '_method' key -- diag uses it to make life easier.
    #
    #     route[ '_method' ] = self.method
    #
    #     # We refer to this route, of course.
    #     route.referenced_by(self[ '_source' ])
    #
    #     # There's a slew of things we'll just copy over transparently; handle
    #     # those.
    #
    #     for key, value in self.items():
    #         if key in Mapping.TransparentRouteKeys:
    #             route[ key ] = value
    #
    #     # Done!
    #     return route


########
## IRMappingGroup is a collection of Mappings. We'll use it to build Envoy routes later,
## so the group itself ends up with some of the group-wide attributes of its Mappings.

class IRMappingGroup (IRResource):
    mappings: List[IRMapping]
    host_redirect: Optional[IRMapping]
    shadow: List[IRMapping]
    group_id: str
    group_weight: List[Union[str, int]]

    # TransparentRouteKeys: ClassVar[Dict[str, bool]] = {
    #     "auto_host_rewrite": True,
    #     "case_sensitive": True,
    #     "envoy_override": True,
    #     "host_rewrite": True,
    #     "path_redirect": True,
    #     "priority": True,
    #     "timeout_ms": True,
    #     "use_websocket": True
    # }

    CoreMappingKeys: ClassVar[Dict[str, bool]] = {
        'group_id': True,
        'headers': True,
        'host_rewrite': True,
        'method': True,
        'prefix': True,
        'prefix_regex': True,
        'rewrite': True,
        'timeout_ms': True
    }

    def __init__(self, ir: 'IR', aconf: Config,
                 location: str,
                 mapping: IRMapping,
                 rkey: str="ir.mappinggroup",
                 kind: str="IRMappingGroup",
                 name: str="ir.mappinggroup",
                 **kwargs) -> None:
        # print("IRMappingGroup __init__ (%s %s %s)" % (kind, name, kwargs))

        if 'host_redirect' in kwargs:
            raise Exception("IRMappingGroup cannot accept a host_redirect as a keyword argument")

        if ('shadow' in kwargs) or ('shadows' in kwargs):
            raise Exception("IRMappingGroup cannot accept shadow or shadows as a keyword argument")

        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, location=location, kind=kind, name=name,
            mappings=[], host_redirect=None, shadows=[], **kwargs
        )

        # Time to lift a bunch of core stuff from the first mapping up into the
        # group.

        if ('group_weight' not in self) and ('route_weight' in mapping):
            self.group_weight = mapping.route_weight

        if ('rewrite' not in self) and ('rewrite' in mapping):
            self.rewrite = mapping.rewrite

        for k in IRMappingGroup.CoreMappingKeys:
            if (k not in self) and (k in mapping):
                self[k] = mapping[k]

        self.add_mapping(aconf, mapping)

    def add_mapping(self, aconf: Config, mapping: IRMapping):
        mismatches = []

        for k in IRMappingGroup.CoreMappingKeys:
            if (k in mapping) and (mapping[k] != self[k]):
                mismatches.append(k)

        if ('rewrite' in mapping) and (mapping.rewrite != self.rewrite):
            mismatches.append('rewrite')

        if mismatches:
            raise Exception("IRMappingGroup %s: cannot accept IRMapping %s with mismatched %s" %
                            (self.group_id, mapping.name, ", ".join(mismatches)))

        host_redirect = mapping.get('host_redirect', False)
        shadow = mapping.get('shadow', False)

        if shadow:
            if self.shadows:
                errstr = "MappingGroup %s: cannot accept %s as second shadow after %s" % \
                         (self.group_id, mapping.name, self.shadows[0].name)
                aconf.post_error(RichStatus.fromError(errstr), resource=self)
            else:
                self.shadows.append(mapping)

                if host_redirect:
                    errstr = "MappingGroup %s: ignoring host_redirect since shadow is set" % self.group_id
                    aconf.post_error(RichStatus.fromError(errstr), resource=self)
        elif host_redirect:
            self.host_redirect = mapping
        else:
            self.mappings.append(mapping)

            if mapping.route_weight > self.group_weight:
                self.group_weight = mapping.group_weight

        self.referenced_by(mapping)

    def add_cluster_for_mapping(self, ir: 'IR', aconf: Config, mapping: IRMapping) -> IRCluster:
        # Find or create the cluster for this Mapping...
        cluster = IRCluster(ir=ir, aconf=aconf,
                            location=mapping.location,
                            service=mapping.service,
                            ctx_name=mapping.get('tls', None),
                            host_rewrite=mapping.get('host_rewrite', False),
                            grpc=mapping.get('grpc', False))

        stored = ir.add_cluster(cluster)
        stored.referenced_by(mapping)

        # ...and return it. Done.
        return stored

    def finalize(self, ir: 'IR', aconf: Config) -> List[IRCluster]:
        """
        Finalize a MappingGroup based on the attributes of its Mappings. Core elements get lifted into
        the Group so we can more easily build Envoy routes; host-redirect and shadow get handled, etc.
        
        :param ir: the IR we're working from 
        :param aconf: the Config we're working from
        :return: a list of the IRClusters this Group uses
        """
        
        # First up,  
        
        total_weight = 0.0
        unspecified_mappings = 0

        rewrite = self.get('rewrite', None)
        if not rewrite:
            # If an empty string has been explicitly specified in the mapping, then we do not need to specify rewrite
            # at all
            self.pop('rewrite')
        elif rewrite is None:
            # By default, the prefix is rewritten to /, so e.g., if we map /prefix1/ to the service service1, then
            # http://ambassador.example.com/prefix1/foo/bar would effectively be written to http://service1/foo/bar
            self.rewrite = '/'

        # if mapping.get('prefix_regex', False):
        #     # if `prefix_regex` is true, then use the `prefix` attribute as the envoy's regex
        #     route['regex'] = mapping['prefix']
        # else:
        #     route['prefix'] = mapping['prefix']

        if self.shadows:
            # Only one shadow is supported right now.
            shadow = self.shadows[0]

            # The shadow is an IRMapping. Save the cluster for it.
            shadow.cluster = self.add_cluster_for_mapping(ir, aconf, shadow)

        # We don't need a cluster for host_redirect: it's just a name to redirect to.

        # if not host_redirect and not shadow:
        #     route['clusters'] = [ { "name": cluster_name,
        #                             "weight": self.get("weight", None) } ]
        # else:
        #     route.setdefault('clusters', [])

        # add_request_headers = self.get('add_request_headers')
        # if add_request_headers:
        #     route[ 'request_headers_to_add' ] = [ ]
        #     for key, value in add_request_headers.items():
        #         route[ 'request_headers_to_add' ].append({"key": key, "value": value})
        #
        # envoy_cors = self.generate_route_cors()
        # if envoy_cors:
        #     route[ 'cors' ] = envoy_cors
        #
        # rate_limits = self.get('rate_limits')
        #
        # if rate_limits:
        #     route[ 'rate_limits' ] = [ ]
        #     for rate_limit in rate_limits:
        #         rate_limits_actions = [
        #             {'type': 'source_cluster'},
        #             {'type': 'destination_cluster'},
        #             {'type': 'remote_address'}
        #         ]
        #
        #         rate_limit_descriptor = rate_limit.get('descriptor', None)
        #
        #         if rate_limit_descriptor:
        #             rate_limits_actions.append({'type': 'generic_key',
        #                                         'descriptor_value': rate_limit_descriptor})
        #
        #         rate_limit_headers = rate_limit.get('headers', [ ])
        #
        #         for rate_limit_header in rate_limit_headers:
        #             rate_limits_actions.append({'type': 'request_headers',
        #                                         'header_name': rate_limit_header,
        #                                         'descriptor_key': rate_limit_header})
        #
        #         route[ 'rate_limits' ].append({'actions': rate_limits_actions})

        # If this is a websocket group, it will support only one cluster right now.
        if self.get('use_websocket', False):
            if len(self.mappings) > 1:
                errmsg = "Only one mapping in a group is supported for websockets; using %s" % \
                         self.mappings[0].name
                self.post_error(RichStatus.fromError(errmsg))

        if not self.host_redirect:
            for mapping in self.mappings:
                mapping.cluster = self.add_cluster_for_mapping(ir, aconf, mapping)

                # Next, does this mapping have a weight assigned?
                if not mapping.get('weight', 0):
                    unspecified_mappings += 1
                else:
                    total_weight += mapping.weight

            # OK, once that's done normalize all the weights.
            if unspecified_mappings:
                for mapping in self.mappings:
                    if not mapping.get("weight", 0):
                        mapping.weight = (100.0 - total_weight)/unspecified_mappings
            elif total_weight != 100.0:
                for mapping in self.mappings:
                    mapping.weight *= 100.0/total_weight

            return list([ mapping.cluster for mapping in self.mappings ])
        else:
            return []

    def as_dict(self) -> Dict:
        od: Dict[str, Any] = {}

        for k in self.keys():
            if (k == 'apiVersion') or (k == 'logger') or (k == 'serialization') or (k == 'ir'):
                continue
            elif k == '_referenced_by':
                refd_by = sorted([ "%s: %s" % (k, self._referenced_by[k].location)
                                   for k in self._referenced_by.keys() ])

                od['_referenced_by'] = refd_by
            elif k == 'rkey':
                od['_rkey'] = self[k]
            elif isinstance(self[k], IRResource):
                od[k] = self[k].as_dict()
            elif k == 'mappings':
                mapping_dicts = reversed(sorted([ x.as_dict() for x in self.mappings ],
                                                key=lambda x: x['route_weight']))
                od[k] = list(mapping_dicts)
            elif k == 'shadows':
                od[k] = list([ x.as_dict() for x in self[k] ])
            elif self[k] is not None:
                od[k] = self[k]

        # print("returning %s" % repr(od))
        return od


class MappingFactory:
    @classmethod
    def load_all(cls, ir: 'IR', aconf: Config) -> None:
        config_info = aconf.get_config("mappings")

        if not config_info:
            return

        assert(len(config_info) > 0)    # really rank paranoia on my part...

        for config in config_info.values():
            # ir.logger.debug("creating mapping for %s" % repr(config))

            mapping = IRMapping(ir, aconf, **config)
            ir.add_mapping(aconf, mapping)

    @classmethod
    def finalize(cls, ir: 'IR', aconf: Config) -> None:
        # OK. We've created whatever IRMappings we need. Time to create the clusters
        # they need.

        # print("CLUSTERS")
        for group in ir.groups.values():
            # print("\n  %s (%s, %s):" % (group.name, group.group_id, group.group_weight))

            group.finalize(ir, aconf)

            # hdrstring = ""
            #
            # if group.headers:
            #     hdrstring = " [ %s ]" % ",".join([ "%s%s%s" % (h.name, "~" if h.regex else "=", h.value)
            #                                        for h in group.headers ])
            #
            # print("    %s %s%s:" % (group.get('method', 'ANY'), group.prefix, hdrstring))
            #
            # mappings = reversed(sorted(group.mappings, key=lambda x: x['route_weight']))
            #
            # for mapping in mappings:
                # print("      %d%% => %s rewrite %s" % (mapping.weight, mapping.service, mapping.rewrite))
                #
                # ctx = mapping.cluster.get('tls_context', None)
                # ctx_name = ctx.name if ctx else "(none)"
                #
                # print("        cluster %s: TLS %s" % (mapping.cluster.name, ctx_name))
                # print("          refs %s" % ", ".join(mapping.cluster._referenced_by))
                # print("          urls %s" % ", ".join(mapping.cluster.urls))
