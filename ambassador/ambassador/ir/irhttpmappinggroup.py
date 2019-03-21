from ambassador.utils import RichStatus
from typing import Any, ClassVar, Dict, List, Optional, Tuple, Union, TYPE_CHECKING
from typing import cast as typecast

from ..config import Config

from .irresource import IRResource
from .ircluster import IRCluster
from .irbasemappinggroup import IRBaseMappingGroup
from .irbasemapping import IRBaseMapping

if TYPE_CHECKING:
    from .ir import IR


########
## IRHTTPMappingGroup is a collection of Mappings. We'll use it to build Envoy routes later,
## so the group itself ends up with some of the group-wide attributes of its Mappings.

class IRHTTPMappingGroup (IRBaseMappingGroup):
    mappings: List[IRBaseMapping]
    host_redirect: Optional[IRBaseMapping]
    shadow: List[IRBaseMapping]
    group_id: str
    group_weight: List[Union[str, int]]
    rewrite: str
    add_request_headers: Dict[str, str]
    add_response_headers: Dict[str, str]
    labels: Dict[str, Any]

    CoreMappingKeys: ClassVar[Dict[str, bool]] = {
        'group_id': True,
        'headers': True,
        'host_rewrite': True,
        # 'labels' doesn't appear in the TransparentKeys list for IRMapping, but it's still
        # a CoreMappingKey -- if it appears, it can't have multiple values within an IRHTTPMappingGroup.
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
                 mapping: IRBaseMapping,
                 rkey: str="ir.mappinggroup",
                 kind: str="IRHTTPMappingGroup",
                 name: str="ir.mappinggroup",
                 **kwargs) -> None:
        # print("IRHTTPMappingGroup __init__ (%s %s %s)" % (kind, name, kwargs))
        del rkey    # silence unused-variable warning

        if 'host_redirect' in kwargs:
            raise Exception("IRHTTPMappingGroup cannot accept a host_redirect as a keyword argument")

        if 'path_redirect' in kwargs:
            raise Exception("IRHTTPMappingGroup cannot accept a path_redirect as a keyword argument")

        if ('shadow' in kwargs) or ('shadows' in kwargs):
            raise Exception("IRHTTPMappingGroup cannot accept shadow or shadows as a keyword argument")

        super().__init__(
            ir=ir, aconf=aconf, rkey=mapping.rkey, location=location, kind=kind, name=name,
            mappings=[], host_redirect=None, shadows=[], **kwargs
        )

        self.add_dict_helper('mappings', IRHTTPMappingGroup.helper_mappings)
        self.add_dict_helper('shadows', IRHTTPMappingGroup.helper_shadows)

        # Time to lift a bunch of core stuff from the first mapping up into the
        # group.

        if ('group_weight' not in self) and ('route_weight' in mapping):
            self.group_weight = mapping.route_weight

        for k in IRHTTPMappingGroup.CoreMappingKeys:
            if (k not in self) and (k in mapping):
                self[k] = mapping[k]

        self.add_mapping(aconf, mapping)

        # self.add_request_headers = {}
        # self.add_response_headers = {}
        # self.labels = {}

    def add_mapping(self, aconf: Config, mapping: IRBaseMapping) -> None:
        mismatches = []

        for k in IRHTTPMappingGroup.CoreMappingKeys:
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
                         (mapping.name, typecast(IRBaseMapping, self.host_redirect).name)
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
    def add_cluster_for_mapping(ir: 'IR', aconf: Config, mapping: IRBaseMapping,
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
                            load_balancer=mapping.get('load_balancer', None),
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

        add_request_headers: Dict[str, Any] = {}
        add_response_headers: Dict[str, Any] = {}

        for mapping in sorted(self.mappings, key=lambda m: m.route_weight):
            # if verbose:
            #     self.ir.logger.debug("%s mapping %s" % (self, mapping.as_json()))

            for k in mapping.keys():
                if k.startswith('_') or mapping.skip_key(k) or (k in IRHTTPMappingGroup.DoNotFlattenKeys):
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
        labels: Dict[str, Any] = self.get('labels', None)

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