from ambassador.utils import RichStatus
from typing import Any, ClassVar, Dict, List, Optional, Tuple, Union, TYPE_CHECKING

from ..config import Config

from .irresource import IRResource
from .ircluster import IRCluster
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
        # "circuit_breaker": True,
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
        # "outlier_detection": True,
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

        # Include the serialization, too.
        "serialization": True,
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

        # OK. After all that we can compute the group ID...
        self.group_id = self._group_id()

        # ...and the route weight.
        self.route_weight = self._route_weight()

        if ('circuit_breaker' in kwargs) or ('outlier_detection' in kwargs):
            self.post_error(RichStatus.fromError("circuit_breaker and outlier_detection are not supported"))

    # self.ir.logger.debug("%s: GID %s route_weight %s" % (self, self.group_id, self.route_weight))

    def setup(self, ir: 'IR', aconf: Config) -> bool:
        # If we have CORS stuff, normalize it.
        if 'cors' in self:
            self.cors = IRCORS(ir=ir, aconf=aconf, location=self.location, **self.cors)

            if self.cors:
                self.cors.referenced_by(self)
            else:
                return False

        return True

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

    def match_tls_context(self, host: str, ir: 'IR'):
        for context in ir.get_tls_contexts():
            for context_host in context.get('hosts'):
                if context_host == host:
                    ir.logger.info("Matched host {} with TLSContext {}".format(host, context.get('name')))
                    self.sni = True
                    return context
        return None

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

    DoNotFlattenKeys: ClassVar[Dict[str, bool]] = dict(CoreMappingKeys)
    DoNotFlattenKeys.update({
        'cluster': True,
        'host': True,
        'kind': True,
        'location': True,
        'name': True,
        'rkey': True,
        'route_weight': True,
        'service': True,
        'weight': True,
    })

    @staticmethod
    def helper_mappings(res: IRResource, k: str) -> Tuple[str, List[dict]]:
        return k, list(reversed(sorted([ x.as_dict() for x in res.mappings ],
                                         key=lambda x: x['route_weight'])))

    @staticmethod
    def helper_shadows(res: IRResource, k: str) -> Tuple[str, List[dict]]:
        return k, list([ x.as_dict() for x in res[k] ])

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

        if 'path_redirect' in kwargs:
            raise Exception("IRMappingGroup cannot accept a path_redirect as a keyword argument")

        if ('shadow' in kwargs) or ('shadows' in kwargs):
            raise Exception("IRMappingGroup cannot accept shadow or shadows as a keyword argument")

        super().__init__(
            ir=ir, aconf=aconf, rkey=mapping.rkey, location=location, kind=kind, name=name,
            mappings=[], host_redirect=None, shadows=[], **kwargs
        )

        self.add_dict_helper('mappings', IRMappingGroup.helper_mappings)
        self.add_dict_helper('shadows', IRMappingGroup.helper_shadows)

        # Time to lift a bunch of core stuff from the first mapping up into the
        # group.

        if ('group_weight' not in self) and ('route_weight' in mapping):
            self.group_weight = mapping.route_weight

        for k in IRMappingGroup.CoreMappingKeys:
            if (k not in self) and (k in mapping):
                self[k] = mapping[k]

        self.add_mapping(aconf, mapping)

    def add_mapping(self, aconf: Config, mapping: IRMapping):
        mismatches = []

        for k in IRMappingGroup.CoreMappingKeys:
            if (k in mapping) and ((k not in self) or
                                   (mapping[k] != self[k])):
                mismatches.append((k, mapping[k], self.get(k, '-unset-')))

        if mismatches:
            raise Exception("IRMappingGroup %s: cannot accept IRMapping %s with mismatched %s" %
                            (self.name, mapping.name, ", ".join([ "%s: %s != %s" % (x, y, z) for x, y, z in mismatches ])))

        # self.ir.logger.debug("%s: add mapping %s" % (self, mapping.as_json()))

        # Per the schema, host_redirect and shadow are Booleans. They won't be _saved_ as
        # Booleans, though: instead we just save the Mapping that they're a part of.
        host_redirect = mapping.get('host_redirect', False)
        shadow = mapping.get('shadow', False)

        # First things first: if both shadow and host_redirect are set in this Mapping,
        # we're going to let shadow win. Kill the host_redirect part.

        if shadow and host_redirect:
            errstr = "At most one of host_redirect and shadow may be set; ignoring host_redirect"
            aconf.post_error(RichStatus.fromError(errstr), resource=mapping)

            mapping.pop('host_redirect', None)
            mapping.pop('path_redirect', None)

        # OK. Is this a shadow Mapping?
        if shadow:
            # Yup. Make sure that we don't have multiple shadows.
            if self.shadows:
                errstr = "cannot accept %s as second shadow after %s" % \
                         (mapping.name, self.shadows[0].name)
                aconf.post_error(RichStatus.fromError(errstr), resource=self)
            else:
                # All good. Save it.
                self.shadows.append(mapping)
        elif host_redirect:
            # Not a shadow, but a host_redirect. Make sure we don't have multiples of
            # those either.

            if self.host_redirect:
                errstr = "cannot accept %s as second host_redirect after %s" % \
                         (mapping.name, self.host_redirect.name)
                aconf.post_error(RichStatus.fromError(errstr), resource=self)
            else:
                # All good. Save it.
                self.host_redirect = mapping
        else:
            # Neither shadow nor host_redirect: save it.
            self.mappings.append(mapping)

            if mapping.route_weight > self.group_weight:
                self.group_weight = mapping.group_weight

        self.referenced_by(mapping)

        # self.ir.logger.debug("%s: group now %s" % (self, self.as_json()))

    def add_cluster_for_mapping(self, ir: 'IR', aconf: Config, mapping: IRMapping,
                                marker: Optional[str] = None) -> IRCluster:
        # Find or create the cluster for this Mapping...
        cluster = IRCluster(ir=ir, aconf=aconf,
                            location=mapping.location,
                            service=mapping.service,
                            ctx_name=mapping.get('tls', None),
                            host_rewrite=mapping.get('host_rewrite', False),
                            grpc=mapping.get('grpc', False),
                            marker=marker)

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

        # verbose = (self.group_id == '2d4a2a00ac0bc25be1b3cebc93b69c73fc9aeaea')
        #
        # if verbose:
        #     self.ir.logger.debug("%s: FINALIZING: %s" % (self, self.as_json()))

        for mapping in sorted(self.mappings, key=lambda m: m.route_weight):
            # if verbose:
            #     self.ir.logger.debug("%s mapping %s" % (self, mapping.as_json()))

            for k in mapping.keys():
                if k.startswith('_') or mapping.skip_key(k) or (k in IRMappingGroup.DoNotFlattenKeys):
                    # if verbose:
                    #     self.ir.logger.debug("%s: don't flatten %s" % (self, k))
                    continue

                # if verbose:
                #     self.ir.logger.debug("%s: flatten %s" % (self, k))

                self[k] = mapping[k]

        # if verbose:
        #     self.ir.logger.debug("%s after flattening %s" % (self, self.as_json()))

        total_weight = 0.0
        unspecified_mappings = 0

        # If no rewrite was given at all, default the rewrite to "/", so /, so e.g., if we map
        # /prefix1/ to the service service1, then http://ambassador.example.com/prefix1/foo/bar
        # would effectively be written to http://service1/foo/bar
        #
        # If they did give a rewrite, leave it alone so that the Envoy config can correctly
        # handle an empty rewrite as no rewriting at all.

        if 'rewrite' not in self:
            self.rewrite = "/"

        if self.shadows:
            # Only one shadow is supported right now.
            shadow = self.shadows[0]

            # The shadow is an IRMapping. Save the cluster for it.
            shadow.cluster = self.add_cluster_for_mapping(ir, aconf, shadow, marker='shadow')

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

        if not self.get('host_redirect', None):
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
