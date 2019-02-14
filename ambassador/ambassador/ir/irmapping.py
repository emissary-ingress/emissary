from ambassador.utils import RichStatus
from typing import Any, ClassVar, Dict, List, Optional, Tuple, Union, TYPE_CHECKING
from typing import cast as typecast

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
        "labels": True, # Only supported in v1, handled in setup
        "method": True,
        "method_regex": True,
        "modules": True,
        # "outlier_detection": True,
        "path_redirect": True,
        # Do not include precedence.
        "prefix": True,
        "prefix_regex": True,
        "priority": True,
        "rate_limits": True, # Only supported in v0, handled in setup
        # Do not include regex_headers.
        # Do not include rewrite.
        "service": True,
        "shadow": True,
        "timeout_ms": True,
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
            hosts = context.get('hosts') or []

            for context_host in hosts:
                if context_host == host:
                    ir.logger.info("Matched host {} with TLSContext {}".format(host, context.get('name')))
                    self.sni = True
                    return context

        return None


########
## IRMappingGroup is a collection of Mappings. We'll use it to build Envoy routes later,
## so the group itself ends up with some of the group-wide attributes of its Mappings.

class IRMappingGroup (IRResource):
    mappings: List[IRMapping]
    host_redirect: Optional[IRMapping]
    shadow: List[IRMapping]
    group_id: str
    group_weight: List[Union[str, int]]
    rewrite: str

    CoreMappingKeys: ClassVar[Dict[str, bool]] = {
        'group_id': True,
        'headers': True,
        'host_rewrite': True,
        # 'labels' doesn't appear in the TransparentKeys list for IRMapping, but it's still
        # a CoreMappingKey -- if it appears, it can't have multiple values within an IRMappingGroup.
        'labels': True,
        'method': True,
        'prefix': True,
        'prefix_regex': True,
        'rewrite': True,
        'timeout_ms': True,
        'bypass_auth': True
    }

    DoNotFlattenKeys: ClassVar[Dict[str, bool]] = dict(CoreMappingKeys)
    DoNotFlattenKeys.update({
        'add_request_headers': True,    # do this manually.
        'add_response_headers': True,    # do this manually.
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
        del rkey    # silence unused-variable warning

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

    def add_mapping(self, aconf: Config, mapping: IRMapping) -> None:
        mismatches = []

        for k in IRMappingGroup.CoreMappingKeys:
            if (k in mapping) and ((k not in self) or
                                   (mapping[k] != self[k])):
                mismatches.append((k, mapping[k], self.get(k, '-unset-')))

        if mismatches:
            self.post_error("cannot accept new mapping %s with mismatched %s" % (
                                mapping.name,
                                ", ".join([ "%s: %s != %s" % (x, y, z) for x, y, z in mismatches ])
                            ))
            return

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
                         (mapping.name, typecast(IRMapping, self.host_redirect).name)
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

    @staticmethod
    def add_cluster_for_mapping(ir: 'IR', aconf: Config, mapping: IRMapping,
                                marker: Optional[str] = None) -> IRCluster:
        # Find or create the cluster for this Mapping...
        cluster = IRCluster(ir=ir, aconf=aconf,
                            location=mapping.location,
                            service=mapping.service,
                            ctx_name=mapping.get('tls', None),
                            host_rewrite=mapping.get('host_rewrite', False),
                            enable_ipv4=mapping.get('enable_ipv4', None),
                            enable_ipv6=mapping.get('enable_ipv6', None),
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

        add_request_headers: Dict[str, str] = {}
        add_response_headers: Dict[str, str] = {}

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

            add_request_headers.update(mapping.get('add_request_headers', {}))
            add_response_headers.update(mapping.get('add_response_headers', {}))

        if add_request_headers:
            self.add_request_headers = add_request_headers
        if add_response_headers:
            self.add_response_headers = add_response_headers

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

        # OK. Save some typing with local variables for default labels and our labels...
        default_labels = ir.ambassador_module.default_labels
        labels = self.get('labels', None)

        if not labels:
            # No labels. Use the default label domain to see if we have some valid defaults.
            defaults = ir.ambassador_module.get_default_labels()

            if defaults:
                domain = ir.ambassador_module.get_default_label_domain()

                self.labels = {
                    domain: [
                        {
                            'defaults': defaults
                        }
                    ]
                }
        else:
            # Walk all the domains in our labels, and prepend the defaults, if any.
            # ir.logger.info("%s: labels %s" % (self.as_json(), labels))

            for domain in labels.keys():
                defaults = ir.ambassador_module.get_default_labels(domain)
                ir.logger.debug("%s: defaults %s" % (domain, defaults))

                if defaults:
                    ir.logger.debug("%s: labels %s" % (domain, labels[domain]))

                    for label in labels[domain]:
                        ir.logger.debug("%s: label %s" % (domain, label))

                        lkeys = label.keys()
                        if len(lkeys) > 1:
                            err = RichStatus.fromError("label has multiple entries (%s) instead of just one" %
                                                       lkeys)
                            aconf.post_error(err, self)

                        lkey = list(lkeys)[0]

                        if lkey.startswith('v0_ratelimit_'):
                            # Don't prepend defaults, as this was imported from a V0 rate_limit.
                            continue

                        label[lkey] = defaults + label[lkey]

        if self.shadows:
            # Only one shadow is supported right now.
            shadow = self.shadows[0]

            # The shadow is an IRMapping. Save the cluster for it.
            shadow.cluster = self.add_cluster_for_mapping(ir, aconf, shadow, marker='shadow')

        # We don't need a cluster for host_redirect: it's just a name to redirect to.

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

        for group in ir.groups.values():
            group.finalize(ir, aconf)
