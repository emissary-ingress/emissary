from typing import TYPE_CHECKING, ClassVar, Dict, List, Optional, Tuple

from ..config import Config
from .irbasemapping import IRBaseMapping
from .irbasemappinggroup import IRBaseMappingGroup
from .ircluster import IRCluster
from .irresource import IRResource

if TYPE_CHECKING:
    from .ir import IR  # pragma: no cover


########
## IRTCPMappingGroup is a collection of IRTCPMappings. We'll use it to build Envoy routes later,
## so the group itself ends up with some of the group-wide attributes of its Mappings.


class IRTCPMappingGroup(IRBaseMappingGroup):
    CoreMappingKeys: ClassVar[Dict[str, bool]] = {
        "address": True,
        "circuit_breakers": True,
        "enable_ipv4": True,
        "enable_ipv6": True,
        "group_id": True,
        "host": True,
        "idle_timeout_ms": True,
        # 'labels' doesn't appear in the TransparentKeys list for IRMapping, but it's still
        # a CoreMappingKey -- if it appears, it can't have multiple values within an IRTCPMappingGroup.
        "labels": True,
        "port": True,
        "tls": True,
    }

    DoNotFlattenKeys: ClassVar[Dict[str, bool]] = dict(CoreMappingKeys)
    DoNotFlattenKeys.update(
        {
            "cluster": True,
            "cluster_key": True,
            "kind": True,
            "location": True,
            "name": True,
            "rkey": True,
            "route_weight": True,
            "service": True,
            "weight": True,
        }
    )

    @staticmethod
    def helper_mappings(res: IRResource, k: str) -> Tuple[str, List[dict]]:
        return k, list(
            reversed(sorted([x.as_dict() for x in res.mappings], key=lambda x: x["route_weight"]))
        )

    def __init__(
        self,
        ir: "IR",
        aconf: Config,
        location: str,
        mapping: IRBaseMapping,
        rkey: str = "ir.mappinggroup",
        kind: str = "IRTCPMappingGroup",
        name: str = "ir.mappinggroup",
        **kwargs,
    ) -> None:
        # print("IRTCPMappingGroup __init__ (%s %s %s)" % (kind, name, kwargs))
        del rkey  # silence unused-variable warning

        super().__init__(
            ir=ir, aconf=aconf, rkey=mapping.rkey, location=location, kind=kind, name=name, **kwargs
        )

        self.add_dict_helper("mappings", IRTCPMappingGroup.helper_mappings)

        # Time to lift a bunch of core stuff from the first mapping up into the
        # group.

        if ("group_weight" not in self) and ("route_weight" in mapping):
            self.group_weight = mapping.route_weight

        for k in IRTCPMappingGroup.CoreMappingKeys:
            if (k not in self) and (k in mapping):
                self[k] = mapping[k]

        self.add_mapping(aconf, mapping)

    def add_mapping(self, aconf: Config, mapping: IRBaseMapping) -> None:
        mismatches = []

        for k in IRTCPMappingGroup.CoreMappingKeys:
            if (k in mapping) and ((k not in self) or (mapping[k] != self[k])):
                mismatches.append((k, mapping[k], self.get(k, "-unset-")))

        if mismatches:
            self.post_error(
                "cannot accept new mapping %s with mismatched %s"
                % (mapping.name, ", ".join(["%s: %s != %s" % (x, y, z) for x, y, z in mismatches]))
            )
            return

        self.mappings.append(mapping)

        if mapping.route_weight > self.group_weight:
            self.group_weight = mapping.group_weight

        self.referenced_by(mapping)

        # self.ir.logger.debug("%s: group now %s" % (self, self.as_json()))

    # Deliberately matches IRListener.bind_to()
    def bind_to(self) -> str:
        bind_addr = self.get("address") or Config.envoy_bind_address
        return f"{bind_addr}-{self.port}"

    def add_cluster_for_mapping(
        self, mapping: IRBaseMapping, marker: Optional[str] = None
    ) -> IRCluster:
        cluster: Optional[IRCluster] = None

        if mapping.cluster_key:
            # Aha. Is our cluster already in the cache?
            cached_cluster = self.ir.cache_fetch(mapping.cluster_key)

            if cached_cluster is not None:
                # We know a priori that anything in the cache under a cluster key must be
                # an IRCluster, but let's assert that rather than casting.
                assert isinstance(cached_cluster, IRCluster)
                cluster = cached_cluster

                self.ir.logger.debug(
                    f"IRTCPMappingGroup: got Cluster from cache for {mapping.cluster_key}"
                )

        if not cluster:
            # Find or create the cluster for this Mapping...
            cluster = IRCluster(
                ir=self.ir,
                aconf=self.ir.aconf,
                parent_ir_resource=mapping,
                location=mapping.location,
                service=mapping.service,
                resolver=mapping.resolver,
                ctx_name=mapping.get("tls", None),
                host_rewrite=mapping.get("host_rewrite", False),
                enable_ipv4=mapping.get("enable_ipv4", None),
                enable_ipv6=mapping.get("enable_ipv6", None),
                circuit_breakers=mapping.get("circuit_breakers", None),
                marker=marker,
                stats_name=self.get("stats_name", None),
            )

        # Make sure that the cluster is really in our IR...
        stored = self.ir.add_cluster(cluster)
        stored.referenced_by(mapping)

        # ...and then check if we just synthesized this cluster.
        if not mapping.cluster_key:
            # Yes. The mapping is already in the cache, but we need to cache the cluster...
            self.ir.cache_add(stored)

            # ...and link the Group to the cluster.
            #
            # Right now, I'm going for maximum safety, which means a single chain linking
            # Mapping -> Group -> Cluster. That means that deleting a single Mapping deletes
            # the Group to which that Mapping is attached, which in turn deletes all the
            # Clusters for that Group.
            #
            # Performance might dictate linking Mapping -> Group and Mapping -> Cluster, so
            # that deleting a Mapping deletes the Group but only the single Cluster. Needs
            # testing.

            self.ir.cache_link(self, stored)

            # Finally, save the cluster's cache_key in this Mapping.
            mapping.cluster_key = stored.cache_key

        # Finally, return the stored cluster. Done.
        return stored

    def finalize(self, ir: "IR", aconf: Config) -> List[IRCluster]:
        """
        Finalize a MappingGroup based on the attributes of its Mappings. Core elements get lifted into
        the Group so we can more easily build Envoy routes; host-redirect and shadow get handled, etc.

        :param ir: the IR we're working from
        :param aconf: the Config we're working from
        :return: a list of the IRClusters this Group uses
        """

        metadata_labels: Dict[str, str] = {}

        for mapping in sorted(self.mappings, key=lambda m: m.route_weight):
            self.ir.logger.debug("%s mapping %s" % (self, mapping.as_json()))

            for k in mapping.keys():
                if (
                    k.startswith("_")
                    or mapping.skip_key(k)
                    or (k in IRTCPMappingGroup.DoNotFlattenKeys)
                ):
                    # self.ir.logger.debug("%s: don't flatten %s" % (self, k))
                    continue

                # self.ir.logger.debug("%s: flatten %s" % (self, k))

                self[k] = mapping[k]

            # Should we have higher weights win over lower if there are conflicts?
            # Should we disallow conflicts?
            metadata_labels.update(mapping.get("metadata_labels") or {})

        if metadata_labels:
            self.metadata_labels = metadata_labels

        # self.ir.logger.debug("%s after flattening %s" % (self, self.as_json()))

        total_weight = 0.0
        unspecified_mappings = 0

        # # OK. Save some typing with local variables for default labels and our labels...
        # labels: Dict[str, Any] = self.get('labels', None)
        #
        # if not labels:
        #     # No labels. Use the default label domain to see if we have some valid defaults.
        #     defaults = ir.ambassador_module.get_default_labels()
        #
        #     if defaults:
        #         domain = ir.ambassador_module.get_default_label_domain()
        #
        #         self.labels = {
        #             domain: [
        #                 {
        #                     'defaults': defaults
        #                 }
        #             ]
        #         }
        # else:
        #     # Walk all the domains in our labels, and prepend the defaults, if any.
        #     # ir.logger.info("%s: labels %s" % (self.as_json(), labels))
        #
        #     for domain in labels.keys():
        #         defaults = ir.ambassador_module.get_default_labels(domain)
        #         ir.logger.debug("%s: defaults %s" % (domain, defaults))
        #
        #         if defaults:
        #             ir.logger.debug("%s: labels %s" % (domain, labels[domain]))
        #
        #             for label in labels[domain]:
        #                 ir.logger.debug("%s: label %s" % (domain, label))
        #
        #                 lkeys = label.keys()
        #                 if len(lkeys) > 1:
        #                     err = RichStatus.fromError("label has multiple entries (%s) instead of just one" %
        #                                                lkeys)
        #                     aconf.post_error(err, self)
        #
        #                 lkey = list(lkeys)[0]
        #
        #                 if lkey.startswith('v0_ratelimit_'):
        #                     # Don't prepend defaults, as this was imported from a V0 rate_limit.
        #                     continue
        #
        #                 label[lkey] = defaults + label[lkey]

        for mapping in self.mappings:
            mapping.cluster = self.add_cluster_for_mapping(mapping, mapping.cluster_tag)

        self.logger.debug(f"Normalizing weights in mappings now...")
        if not self.normalize_weights_in_mappings():
            self.post_error(f"Could not normalize mapping weights, ignoring...")
            return []

        return list([mapping.cluster for mapping in self.mappings])
