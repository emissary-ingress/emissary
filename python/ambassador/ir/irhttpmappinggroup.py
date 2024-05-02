import re
from typing import TYPE_CHECKING, Any, ClassVar, Dict, List, Optional, Tuple, Union
from typing import cast
from typing import cast as typecast

from ambassador.utils import RichStatus

from ..config import Config
from .irbasemapping import IRBaseMapping
from .irbasemappinggroup import IRBaseMappingGroup
from .ircluster import IRCluster
from .irhttpmapping import IRHTTPMapping
from .irresource import IRResource
from .irutils import are_mapping_group_fixes_disabled

if TYPE_CHECKING:
    from .ir import IR  # pragma: no cover


########
## IRHTTPMappingGroup is a collection of Mappings. We'll use it to build Envoy routes later.
## The group itself shares a common group of settings for matching traffic. All mappings added to the group
## should have the same traffic matching settings. Settings for traffic modifications and where traffic is routed
## are set on a per-mapping basis.


class IRHTTPMappingGroup(IRBaseMappingGroup):
    shadow_mappings: List[IRHTTPMapping]

    # This is the initial mapping used to create the group. We keep it in right now to support the case when
    # ENABLE_MAPPING_GROUP_FIXES is not true and we want to have a single set of settings (adding headers, etc.)
    # be shared across all mappings in a group. This will be removed in a future release.
    seed_mapping: IRHTTPMapping

    # List of the fields within Mappings that control what requests to match on.
    # we are not adding these as class fields since we do not want these keys to be set at all unless
    # the mappings that are added to this group use those settings
    TrafficMatchSettings: ClassVar[Dict[str, bool]] = {
        "host": True,
        "host_regex": True,
        "prefix": True,
        "prefix_exact": True,
        "prefix_regex": True,
        "case_sensitive": True,
        "headers": True,
        "method": True,
        "method_regex": True,
        "query_parameters": True,
        "precedence": True,
    }

    CoreMappingKeys: ClassVar[Dict[str, bool]] = {
        "bypass_auth": True,
        "bypass_error_response_overrides": True,
        "circuit_breakers": True,
        "cluster_timeout_ms": True,
        "connect_timeout_ms": True,
        "cluster_idle_timeout_ms": True,
        "cluster_max_connection_lifetime_ms": True,
        "group_id": True,
        "headers": True,
        # 'host_rewrite': True,
        # 'idle_timeout_ms': True,
        "keepalive": True,
        # 'labels' doesn't appear in the TransparentKeys list for IRMapping, but it's still
        # a CoreMappingKey -- if it appears, it can't have multiple values within an IRHTTPMappingGroup.
        "labels": True,
        "load_balancer": True,
        # 'metadata_labels' will get flattened by merging. The group gets all the labels that all its
        # Mappings have.
        "method": True,
        "prefix": True,
        "prefix_regex": True,
        "prefix_exact": True,
        # 'rewrite': True,
        # 'timeout_ms': True
    }

    # We don't flatten cluster_key and stats_name because the whole point of those
    # two is that you're asking for something special with stats. Note that we also
    # don't do collision checking specially for the stats_name: if you ask for the
    # same stats_name in two unrelated mappings, on your own head be it.

    DoNotFlattenKeys: ClassVar[Dict[str, bool]] = dict(CoreMappingKeys)
    DoNotFlattenKeys.update(
        {
            "add_request_headers": True,  # do this manually.
            "add_response_headers": True,  # do this manually.
            "cluster": True,
            "cluster_key": True,  # See above about stats.
            "kind": True,
            "location": True,
            "name": True,
            "resolver": True,  # can't flatten the resolver...
            "rkey": True,
            "route_weight": True,
            "service": True,
            "stats_name": True,  # See above about stats.
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
        mapping: IRHTTPMapping,
        rkey: str = "ir.mappinggroup",
        kind: str = "IRHTTPMappingGroup",
        name: str = "ir.mappinggroup",
        **kwargs,
    ) -> None:
        del rkey  # silence unused-variable warning

        self.shadow_mappings: List[IRHTTPMapping] = []

        super().__init__(
            ir=ir, aconf=aconf, rkey=mapping.rkey, location=location, kind=kind, name=name, **kwargs
        )
        self.add_dict_helper("mappings", IRHTTPMappingGroup.helper_mappings)

        if ("group_weight" not in self) and ("route_weight" in mapping):
            self.group_weight = mapping.route_weight

        # Time to lift the traffic matching settings from the first mapping up into the group
        for k in IRHTTPMappingGroup.TrafficMatchSettings:
            if k in mapping:
                self[k] = mapping[k]
        if "group_id" in mapping:
            self["group_id"] = mapping["group_id"]

        self.seed_mapping = mapping
        self.add_mapping(aconf, mapping)

    def add_mapping(self, aconf: Config, mapping: IRHTTPMapping) -> None:
        mismatches = []

        if are_mapping_group_fixes_disabled():
            for k in IRHTTPMappingGroup.CoreMappingKeys:
                if (k in mapping) and (
                    (k not in self.seed_mapping) or (mapping[k] != self.seed_mapping[k])
                ):
                    mismatches.append((k, mapping[k], self.get(k, "-unset-")))
                else:
                    for k in mapping.keys():
                        if (
                            k.startswith("_")
                            or mapping.skip_key(k)
                            or (k in IRHTTPMappingGroup.DoNotFlattenKeys)
                        ):
                            continue
                        self.seed_mapping[k] = mapping[k]
        else:
            for k in IRHTTPMappingGroup.TrafficMatchSettings:
                if (k in mapping) and ((k not in self) or (mapping[k] != self[k])):
                    mismatches.append((k, mapping[k], self.get(k, "-unset-")))

        # First things first: if both shadow and host_redirect are set in this Mapping,
        # we're going to let shadow win. Kill the host_redirect part.

        host_redirect = mapping.get("host_redirect", False)
        shadow = mapping.get("shadow", False)
        if shadow and host_redirect:
            errstr = "At most one of host_redirect and shadow may be set; ignoring host_redirect"
            aconf.post_error(RichStatus.fromError(errstr), resource=mapping)

            mapping.pop("host_redirect", None)
            mapping.pop("path_redirect", None)
            mapping.pop("prefix_redirect", None)
            mapping.pop("regex_redirect", None)

        if mismatches:
            self.post_error(
                "http mapping group cannot accept new mapping %s with mismatched %s."
                "Please verify field is set with the same value in all related mappings."
                "Example: When canary is configured, related mappings should have same request matching fields and values (ex: prefix/hostname)"
                % (mapping.name, ", ".join(["%s: %s != %s" % (x, y, z) for x, y, z in mismatches]))
            )
            return
        else:
            # All good. Save this mapping.
            if shadow:
                self.shadow_mappings.append(mapping)
            else:
                self.mappings.append(mapping)
                if mapping.route_weight > self.group_weight:
                    self.group_weight = mapping.group_weight

        self.referenced_by(mapping)

    def add_cluster_for_mapping(
        self, mapping: IRHTTPMapping, marker: Optional[str] = None
    ) -> IRCluster:
        # Find or create the cluster for this Mapping...

        self.ir.logger.debug(
            f"IRHTTPMappingGroup: {self.group_id} adding cluster for Mapping {mapping.name} (key {mapping.cluster_key})"
        )

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
                    f"IRHTTPMappingGroup: got Cluster from cache for {mapping.cluster_key}"
                )

        if not cluster:
            # OK, we have to actually do some work.
            self.ir.logger.debug(f"IRHTTPMappingGroup: synthesizing Cluster for {mapping.name}")
            cluster = IRCluster(
                ir=self.ir,
                aconf=self.ir.aconf,
                parent_ir_resource=mapping,
                location=mapping.location,
                service=mapping.service,
                resolver=mapping.resolver,
                ctx_name=mapping.get("tls", None),
                dns_type=mapping.get("dns_type", "strict_dns"),
                health_checks=mapping.get("health_checks", None),
                host_rewrite=mapping.get("host_rewrite", False),
                enable_ipv4=mapping.get("enable_ipv4", None),
                enable_ipv6=mapping.get("enable_ipv6", None),
                grpc=mapping.get("grpc", False),
                load_balancer=mapping.get("load_balancer", None),
                keepalive=mapping.get("keepalive", None),
                connect_timeout_ms=mapping.get("connect_timeout_ms", 3000),
                cluster_idle_timeout_ms=mapping.get("cluster_idle_timeout_ms", None),
                cluster_max_connection_lifetime_ms=mapping.get(
                    "cluster_max_connection_lifetime_ms", None
                ),
                circuit_breakers=mapping.get("circuit_breakers", None),
                marker=marker,
                stats_name=mapping.get("stats_name"),
                respect_dns_ttl=mapping.get("respect_dns_ttl", False),
            )

        # Make sure that the cluster is actually in our IR...
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
        self.ir.logger.debug(
            f"IRHTTPMappingGroup: %s returning cluster %s for Mapping %s",
            self.group_id,
            stored,
            mapping.name,
        )
        return stored

    def finalize(self, ir: "IR", aconf: Config) -> List[IRCluster]:
        """
        Finalize a MappingGroup based on the attributes of its Mappings.
        :param ir: the IR we're working from
        :param aconf: the Config we're working from
        :return: a list of the IRClusters this Group uses
        """
        metadata_labels: Dict[str, str] = {}

        self.ir.logger.debug(f"IRHTTPMappingGroup: finalize %s", self.group_id)
        for mapping in sorted(self.mappings, key=lambda m: m.route_weight):
            assert isinstance(mapping, IRHTTPMapping)
            # If no rewrite was given at all, default the rewrite to "/", so /, so e.g., if we map
            # /prefix1/ to the service service1, then http://ambassador.example.com/prefix1/foo/bar
            # would effectively be written to http://service1/foo/bar
            #
            # If they did give a rewrite, leave it alone so that the Envoy config can correctly
            # handle an empty rewrite as no rewriting at all.
            if "rewrite" not in mapping:
                mapping.rewrite = "/"

            if mapping.get("load_balancer", None) is None:
                mapping["load_balancer"] = ir.ambassador_module.load_balancer

            if mapping.get("keepalive", None) is None:
                keepalive_default = ir.ambassador_module.get("keepalive", None)
                if keepalive_default:
                    mapping["keepalive"] = keepalive_default

            labels: Dict[str, Any] = mapping.get("labels", None)

            if not labels:
                # No labels. Use the default label domain to see if we have some valid defaults.
                defaults = ir.ambassador_module.get_default_labels()
                if defaults:
                    domain = ir.ambassador_module.get_default_label_domain()
                    mapping.labels = {domain: [{"defaults": defaults}]}
            else:
                # Walk all the domains in our labels, and prepend the defaults, if any.
                for domain in labels.keys():
                    defaults = ir.ambassador_module.get_default_labels(domain)
                    ir.logger.debug("%s: defaults %s" % (domain, defaults))
                    if defaults:
                        ir.logger.debug("%s: labels %s" % (domain, labels[domain]))
                        for label in labels[domain]:
                            ir.logger.debug("%s: label %s" % (domain, label))
                            lkeys = label.keys()
                            if len(lkeys) > 1:
                                err = RichStatus.fromError(
                                    "label has multiple entries (%s) instead of just one" % lkeys
                                )
                                aconf.post_error(err, self)
                            lkey = list(lkeys)[0]
                            if lkey.startswith("v0_ratelimit_"):
                                # Don't prepend defaults, as this was imported from a V0 rate_limit.
                                continue
                            label[lkey] = defaults + label[lkey]
            metadata_labels.update(mapping.get("metadata_labels") or {})

        if metadata_labels:
            self.metadata_labels = metadata_labels

        for mapping in self.mappings:
            assert isinstance(mapping, IRHTTPMapping)
            # Mappings that do hostname redirects don't need a cluster
            redir = mapping.get("host_redirect", None)
            if not redir:
                mapping.cluster = self.add_cluster_for_mapping(mapping, mapping.cluster_tag)

        for shadow_mapping in self.shadow_mappings:
            # Add a special marker for mappings that do traffic mirroring/shadowing
            shadow_mapping.cluster = self.add_cluster_for_mapping(shadow_mapping, marker="shadow")

        self.ir.logger.debug(f"IRHTTPMappingGroup: normalizing weights for %s", self.group_id)

        normalized_mappings, ok = self.normalize_weights_in_mappings(self.mappings)
        if not ok:
            self.post_error(f"Could not normalize mapping weights, ignoring...")
            return []
        self.mappings = normalized_mappings

        if len(self.shadow_mappings) > 0:
            self.ir.logger.debug(
                f"IRHTTPMappingGroup: normalizing shadow weights for %s", self.group_id
            )

            normalized_mappings, ok = self.normalize_weights_in_mappings(
                cast(List[IRBaseMapping], self.shadow_mappings)
            )
            if not ok:
                self.post_error(f"Could not normalize shadow weights, ignoring...")
                return []

            self.shadow_mappings = cast(List[IRHTTPMapping], normalized_mappings)

        # return all the clusters from our mappings (note that redirect mappings won't have a cluster)
        return [mapping.cluster for mapping in self.mappings if "cluster" in mapping]
