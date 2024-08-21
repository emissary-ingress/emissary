# Copyright 2018 Datawire. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License

import logging
import re
from typing import Any, Dict, List, Optional, Tuple
from typing import cast as typecast

from ..envoy import EnvoyConfig
from ..ir import IR
from ..ir.irbasemappinggroup import IRBaseMappingGroup
from ..ir.irhttpmappinggroup import IRHTTPMappingGroup
from ..utils import dump_json
from .envoy_stats import EnvoyStats


class DiagSource(dict):
    pass


class DiagCluster(dict):
    """
    A DiagCluster represents what Envoy thinks about the health of a cluster.
    DO NOT JUST PASS AN IRCluster into DiagCluster; turn it into a dict with
    .as_dict() first.
    """

    def __init__(self, cluster) -> None:
        super().__init__(**cluster)

    def update_health(self, other: dict) -> None:
        for from_key, to_key in [
            ("health", "_health"),
            ("hmetric", "_hmetric"),
            ("hcolor", "_hcolor"),
        ]:
            if from_key in other:
                self[to_key] = other[from_key]

    def default_missing(self) -> dict:
        for key, default in [
            ("service", "unknown service!"),
            ("weight", 100),
            ("_hmetric", "unknown"),
            ("_hcolor", "orange"),
        ]:
            if not self.get(key, None):
                self[key] = default

        return dict(self)

    @classmethod
    def unknown_cluster(cls):
        return DiagCluster(
            {
                "service": "unknown service!",
                "_health": "unknown cluster!",
                "_hmetric": "unknown",
                "_hcolor": "orange",
            }
        )


class DiagClusters:
    """
    DiagClusters is, unsuprisingly, a set of DiagCluster. The thing about DiagClusters
    is that the [] operator always gives you a valid DiagCluster -- it'll use DiagCluster.unknown()
    to make a new DiagCluster if you ask for one that doesn't exist.
    """

    clusters: Dict[str, DiagCluster]

    def __init__(self, clusters: Optional[List[dict]] = None) -> None:
        self.clusters = {}

        if clusters:
            for cluster in typecast(List[dict], clusters):
                self[cluster["name"]] = DiagCluster(cluster)

    def __getitem__(self, key: str) -> DiagCluster:
        if key not in self.clusters:
            self.clusters[key] = DiagCluster.unknown_cluster()

        return self.clusters[key]

    def __setitem__(self, key: str, value: DiagCluster) -> None:
        self.clusters[key] = value

    def __contains__(self, key: str) -> bool:
        return key in self.clusters

    def as_json(self):
        return dump_json(self.clusters, pretty=True)


class DiagResult:
    """
    A DiagResult is the result of a diagnostics request, whether for an
    overview or for a particular key.
    """

    def __init__(self, diag: "Diagnostics", estat: EnvoyStats, request) -> None:
        self.diag = diag
        self.logger = self.diag.logger
        self.estat = estat

        # Go ahead and grab Envoy cluster stats for all possible clusters.
        # XXX This might be a bit silly.
        self.cstats = {
            cluster.name: self.estat.cluster_stats(cluster.stats_name)
            for cluster in self.diag.clusters.values()
        }

        # Save the request host and scheme. We'll need them later.
        self.request_host = request.headers.get("Host", "*")
        self.request_scheme = request.headers.get("X-Forwarded-Proto", "http").lower()

        # All of these things reflect _only_ resources that are relevant to the request
        # we're handling -- e.g. if you ask for a particular group, you'll only get the
        # clusters that are part of that group.
        self.clusters: Dict[str, DiagCluster] = {}  # Envoy clusters
        self.routes: List[dict] = []  # Envoy routes
        self.element_keys: Dict[str, bool] = {}  # Active element keys
        self.ambassador_resources: Dict[
            str, str
        ] = {}  # Actually serializations of Ambassador config resources
        self.envoy_resources: Dict[str, dict] = {}  # Envoy config resources

    def as_dict(self) -> Dict[str, Any]:
        return {
            "cluster_stats": self.cstats,
            "cluster_info": self.clusters,
            "route_info": self.routes,
            "active_elements": sorted(self.element_keys.keys()),
            "ambassador_resources": self.ambassador_resources,
            "envoy_resources": self.envoy_resources,
        }

    def include_element(self, key: str) -> None:
        """
        Note that a particular key is something relevant to this result -- e.g.
        'oh, the key foo-mapping.1 is active here'.

        One problem here is that we don't currently cycle over to make sure that
        all the requisite higher-level objects are brought in when we mark an
        element active. This needs fixing.

        :param key: the key we want to remember as being active.
        """
        self.element_keys[key] = True

    def include_referenced_elements(self, obj: dict) -> None:
        """
        Include all of the elements in the given object's _referenced_by
        array.

        :param obj: object for which to include referencing keys
        """

        for element_key in obj["_referenced_by"]:
            self.include_element(element_key)

    def include_cluster(self, cluster: dict) -> DiagCluster:
        """
        Note that a particular cluster and everything that references it are
        relevant to this result. If the cluster has related health information in
        our cstats, fold that in too.

        Don't pass an IRCluster here -- turn it into a dict with as_dict()
        first.

        Returns the DiagCluster that we actually use to hold everything.

        :param cluster: dictionary version of a cluster to mark as active.
        :return: the DiagCluster for this cluster
        """

        c_name = cluster["name"]

        if c_name not in self.clusters:
            self.clusters[c_name] = DiagCluster(cluster)

        if c_name in self.cstats:
            self.clusters[c_name].update_health(self.cstats[c_name])

        self.include_referenced_elements(cluster)

        return self.clusters[c_name]

    def include_httpgroup(self, group: IRHTTPMappingGroup) -> None:
        """
        Note that a particular IRHTTPMappingGroup, all of the clusters it uses for upstream
        traffic, and everything that references it are relevant to this result.

        This method actually does a fair amount of work around handling clusters, shadow
        clusters, and host_redirects. It would be a horrible mistake to duplicate this
        elsewhere.

        :param group: IRHTTPMappingGroup to include
        """

        # self.logger.debug("GROUP %s" % group.as_json())

        prefix = group["prefix"] if "prefix" in group else group["regex"]
        rewrite = group.get("rewrite", "/")
        method = "*"
        host = None

        route_clusters: List[DiagCluster] = []

        for mapping in group.get("mappings", []):
            cluster = mapping["cluster"]

            mapping_cluster = self.include_cluster(cluster.as_dict())
            mapping_cluster.update({"weight": mapping.get("weight", 100)})

            # self.logger.debug("GROUP %s CLUSTER %s %d%% (%s)" %
            #                   (group['group_id'], c_name, mapping['weight'], mapping_cluster))

            route_clusters.append(mapping_cluster)

        host_redir = group.get("host_redirect", None)

        if host_redir:
            # XXX Stupid hackery here. redirect_cluster should be a real
            # IRCluster object.
            redirect_cluster = self.include_cluster(
                {
                    "name": host_redir["name"],
                    "service": host_redir["service"],
                    "weight": 100,
                    "type_label": "redirect",
                    "_referenced_by": [host_redir["rkey"]],
                }
            )

            route_clusters.append(redirect_cluster)

            self.logger.debug("host_redirect route: %s" % group)
            self.logger.debug("host_redirect cluster: %s" % redirect_cluster)

        shadows = group.get("shadows", [])

        for shadow in shadows:
            # Shadows have a real cluster object.
            shadow_dict = shadow["cluster"].as_dict()
            shadow_dict["type_label"] = "shadow"

            shadow_cluster = self.include_cluster(shadow_dict)
            route_clusters.append(shadow_cluster)

            self.logger.debug("shadow route: %s" % group)
            self.logger.debug("shadow cluster: %s" % shadow_cluster)

        headers = []

        for header in group.get("headers", []):
            hdr_name = header.get("name", None)
            hdr_value = header.get("value", None)

            if hdr_name == ":authority":
                host = hdr_value
            elif hdr_name == ":method":
                method = hdr_value
            else:
                headers.append(header)

        sep = "" if prefix.startswith("/") else "/"
        route_key = "%s://%s%s%s" % (
            self.request_scheme,
            host if host else self.request_host,
            sep,
            prefix,
        )

        route_info = {
            "_route": group.as_dict(),
            "_source": group["location"],
            "_group_id": group["group_id"],
            "key": route_key,
            "prefix": prefix,
            "rewrite": rewrite,
            "method": method,
            "headers": headers,
            "clusters": [x.default_missing() for x in route_clusters],
            "host": host if host else "*",
        }

        if "precedence" in group:
            route_info["precedence"] = group["precedence"]

        metadata_labels = group.get("metadata_labels") or {}
        diag_class = metadata_labels.get("ambassador_diag_class") or None

        if diag_class:
            route_info["diag_class"] = diag_class

        self.routes.append(route_info)
        self.include_referenced_elements(group)

    def finalize(self) -> None:
        """
        Make sure that all the elements we've marked as included actually appear
        in the ambassador_resources and envoy_resources dictionaries, so that the
        UI can properly connect all the dots.
        """

        for key in self.element_keys.keys():
            amb_el_info = self.diag.ambassador_elements.get(key, None)

            if amb_el_info:
                serialization = amb_el_info.get("serialization", None)

                if serialization:
                    self.ambassador_resources[key] = serialization

                # What about errors?

            # Also make sure we have Envoy outputs for these things.
            envoy_el_info = self.diag.envoy_elements.get(key, None)

            if envoy_el_info:
                self.envoy_resources[key] = envoy_el_info


class Diagnostics:
    """
    Information needed by the Diagnostics UI. This has to be instantiated
    from an IR and an EnvoyConfig (it doesn't matter which version).

    The flow here is:

    - create the Diagnostics object
    - call the .overview method to get a DiagResult that has an overview of
      the whole Ambassador setup, or
    - call the .lookup method to get a DiagResult that zeroes in on a particular
      chunk of the world (like a group, or a particular rkey, etc.)
    """

    ir: IR
    econf: EnvoyConfig
    estats: Optional[EnvoyStats]

    source_map: Dict[str, Dict[str, bool]]

    reKeyIndex = re.compile(r"\.(\d+)$")

    filter_map = {"IRAuth": "AuthService", "IRRateLimit": "RateLimitService"}

    def __init__(self, ir: IR, econf: EnvoyConfig) -> None:
        self.logger = logging.getLogger("ambassador.diagnostics")
        self.logger.debug("---- building diagnostics")

        self.ir = ir
        self.econf = econf
        self.estats = None

        # A fully-qualified key is e.g. "ambassador.yaml.1" -- source location plus
        # object index. An unqualified key is something like "ambassador.yaml" -- no
        # index.
        #
        # self.source_map permits us to look up any (potentially unqualified) key
        # and find a list of fully-qualified keys contained in the key we looked
        # up.
        #
        # self.ambassador_elements has the incoming Ambassador configuration resources,
        # indexed by fully-qualified key.
        #
        # self.envoy_elements has the created Envoy configuration resources, indexed
        # by fully-qualified key.

        self.source_map: Dict[str, Dict[str, bool]] = {}
        self.ambassador_elements: Dict[str, dict] = {}
        self.envoy_elements: Dict[str, dict] = {}
        self.ambassador_services: List[dict] = []
        self.ambassador_resolvers: List[dict] = []

        # Warn people about upcoming deprecations.

        warn_auth = False
        warn_ratelimit = False

        for filter in self.ir.filters:
            if filter.kind == "IRAuth":
                proto = filter.get("proto") or "http"

                if proto.lower() != "http":
                    warn_auth = True

            if filter.kind == "IRRateLimit":
                warn_ratelimit = True

        things_to_warn = []

        if warn_auth:
            things_to_warn.append("AuthServices")

        if warn_ratelimit:
            things_to_warn.append("RateLimitServices")

        if things_to_warn:
            self.ir.aconf.post_notice(
                f'A future Ambassador version will change the GRPC protocol version for {" and ".join(things_to_warn)}. See the CHANGELOG for details.'
            )

        # # Warn people about the default port change.
        # if self.ir.ambassador_module.service_port < 1024:
        #     # Does it look like they explicitly asked for this?
        #     amod = self.ir.aconf.get_module('ambassador')
        #
        #     if not (amod and amod.get('service_port')):
        #         # They did not explictly set the port. Warn them about the
        #         # port change.
        #         new_defaults = [ "port 8080 for HTTP" ]
        #
        #         if self.ir.tls_contexts:
        #             new_defaults.append("port 8443 for HTTPS")
        #
        #         default_ports = " and ".join(new_defaults)
        #
        #         listen_ports = [ str(l.service_port) for l in self.ir.listeners ]
        #         self.ir.logger.info("listen_ports %s" % listen_ports)
        #
        #         port_or_ports = "port" if (len(listen_ports) == 1) else "ports"
        #
        #         last_port = listen_ports.pop()
        #
        #         els = [ last_port ]
        #
        #         if len(listen_ports) > 0:
        #             els.insert(0, ", ".join(listen_ports))
        #
        #         port_nums = " and ".join(els)
        #
        #         m1 = f'Ambassador 0.60 will default to listening on {default_ports}.'
        #         m2 = f'You will need to change your configuration to continue using {port_or_ports} {port_nums}.'
        #
        #         self.ir.aconf.post_notice(f'{m1} {m2}')

        # Copy in the toplevel 'error' and 'notice' sets.
        self.errors = self.ir.aconf.errors
        self.notices = self.ir.aconf.notices

        # Next up, walk the list of Ambassador sources.
        for key, rsrc in self.ir.aconf.sources.items():
            uqkey = key  # Unqualified key, e.g. ambassador.yaml
            fqkey = uqkey  # Fully-qualified key, e.g. ambassador.yaml.1

            key_index = None

            if "rkey" in rsrc:
                uqkey, key_index = self.split_key(rsrc.rkey)

            if key_index is not None:
                fqkey = "%s.%s" % (uqkey, key_index)

            location, _ = self.split_key(rsrc.get("location", key))

            self.logger.debug(
                "  %s (%s): UQ %s, FQ %s, LOC %s" % (key, rsrc, uqkey, fqkey, location)
            )

            self.remember_source(uqkey, fqkey, location, rsrc.rkey)

            ambassador_element: dict = self.ambassador_elements.setdefault(
                fqkey, {"location": location, "kind": rsrc.kind}
            )

            if uqkey and (uqkey != fqkey):
                ambassador_element["parent"] = uqkey

            serialization = rsrc.get("serialization", None)
            if serialization:
                if ambassador_element["kind"] == "Secret":
                    serialization = "kind: Secret\ndata: (elided by Ambassador)\n"
                ambassador_element["serialization"] = serialization

        # Next up, the Envoy elements.
        for kind, elements in self.econf.elements.items():
            for fqkey, envoy_element in elements.items():
                # The key here should already be fully qualified.
                uqkey, _ = self.split_key(fqkey)

                element_dict = self.envoy_elements.setdefault(fqkey, {})
                element_list = element_dict.setdefault(kind, [])
                element_list.append({k: v for k, v in envoy_element.items() if k[0] != "_"})

        # Always generate the full group set so that we can look up groups.
        self.groups = {
            "grp-%s" % group.group_id: group
            for group in self.ir.groups.values()
            if group.location != "--diagnostics--"
        }

        # Always generate the full cluster set so that we can look up clusters.
        self.clusters = {
            cluster.name: cluster
            for cluster in self.ir.clusters.values()
            if cluster.location != "--diagnostics--"
        }

        # Build up our Ambassador services too (auth, ratelimit, tracing).
        self.ambassador_services = []

        for filt in self.ir.filters:
            # self.logger.debug("FILTER %s" % filter.as_json())

            if filt.kind in Diagnostics.filter_map:
                type_name = Diagnostics.filter_map[filt.kind]
                self.add_ambassador_service(filt, type_name)

        if self.ir.tracing:
            self.add_ambassador_service(
                self.ir.tracing, "TracingService (%s)" % self.ir.tracing.driver
            )

        self.ambassador_resolvers = []
        used_resolvers: Dict[str, List[str]] = {}

        for group in self.groups.values():
            for mapping in group.mappings:
                resolver_name = mapping.resolver
                group_list = used_resolvers.setdefault(resolver_name, [])
                group_list.append(group.rkey)

        for name, resolver in sorted(self.ir.resolvers.items()):
            if name in used_resolvers:
                self.add_ambassador_resolver(resolver, used_resolvers[name])

    def add_ambassador_service(self, svc, type_name) -> None:
        """
        Remember information about a given Ambassador-wide service (Auth, RateLimit, Tracing).

        :param svc: service record
        :param type_name: what kind of thing is this?
        """

        cluster = svc.cluster
        urls = cluster.urls

        svc_weight = 100.0 / len(urls)

        for url in urls:
            self.ambassador_services.append(
                {
                    "type": type_name,
                    "_source": svc.location,
                    "name": url,
                    "cluster": cluster.name,
                    "_service_weight": svc_weight,
                }
            )

    def add_ambassador_resolver(self, resolver, group_list) -> None:
        """
        Remember information about a given Ambassador-wide resolver.

        :param resolver: resolver record
        :param group_list: list of groups that use this resolver
        """

        self.ambassador_resolvers.append(
            {
                "kind": resolver.kind,
                "_source": resolver.location,
                "name": resolver.name,
                "groups": group_list,
            }
        )

    @staticmethod
    def split_key(key) -> Tuple[str, Optional[str]]:
        """
        Split a key into its components (the base name and the object index).

        :param key: possibly-qualified key
        :return: tuple of the base and a possible index
        """

        key_base = key
        key_index = None

        m = Diagnostics.reKeyIndex.search(key)

        if m:
            key_base = key[: m.start()]
            key_index = m.group(1)

        return key_base, key_index

    def as_dict(self) -> dict:
        return {
            "source_map": self.source_map,
            "ambassador_services": self.ambassador_services,
            "ambassador_resolvers": self.ambassador_resolvers,
            "ambassador_elements": self.ambassador_elements,
            "envoy_elements": self.envoy_elements,
            "errors": self.errors,
            "notices": self.notices,
            "groups": {key: self.flattened(value) for key, value in self.groups.items()},
            # 'clusters': { key: value.as_dict() for key, value in self.clusters.items() },
            "tlscontexts": [x.as_dict() for x in self.ir.tls_contexts.values()],
        }

    def flattened(self, group: IRBaseMappingGroup) -> dict:
        flattened = {k: v for k, v in group.as_dict().items() if k != "mappings"}
        flattened_mappings = []

        for m in group["mappings"]:
            fm = {
                "_active": m["_active"],
                "_errored": m["_errored"],
                "_rkey": m["rkey"],
                "location": m["location"],
                "name": m["name"],
                "cluster_service": m.get("cluster", {}).get("service"),
                "cluster_name": m.get("cluster", {}).get("envoy_name"),
            }

            if flattened["kind"] == "IRHTTPMappingGroup":
                fm["prefix"] = m.get("prefix")

            rewrite = m.get("rewrite", None)

            if rewrite:
                fm["rewrite"] = rewrite

            host = m.get("host", None)

            if host:
                fm["host"] = host

            flattened_mappings.append(fm)

        flattened["mappings"] = flattened_mappings

        return flattened

    def _remember_source(self, src_key: str, dest_key: str) -> None:
        """
        Link keys of active sources together. The source map lets us answer questions
        like 'which objects does ambassador.yaml define?' and this is the primitive
        that actually populates the map.

        The src_key is where you start the lookup; the dest_key is something defined
        by the src_key. They can be the same.

        :param src_key: the starting key (ambassador.yaml)
        :param dest_key: the destination key (ambassador.yaml.1)
        """

        src_map = self.source_map.setdefault(src_key, {})
        src_map[dest_key] = True

    def remember_source(
        self, uqkey: str, fqkey: Optional[str], location: Optional[str], dest_key: str
    ) -> None:
        """
        Populate the source map in various ways. A mapping from uqkey to dest_key is
        always added; mappings for fqkey and location are added if they are unique
        keys.

        :param uqkey: unqualified source key
        :param fqkey: qualified source key
        :param location: source location
        :param dest_key: key of object being defined
        """
        self._remember_source(uqkey, dest_key)

        if fqkey and (fqkey != uqkey):
            self._remember_source(fqkey, dest_key)

        if location and (location != uqkey) and (location != fqkey):
            self._remember_source(location, dest_key)

    def overview(self, request, estat: EnvoyStats) -> Dict[str, Any]:
        """
        Generate overview data describing the whole Ambassador setup, most
        notably the routing table. Returns the dictionary form of a DiagResult.

        :param request: the Flask request being handled
        :param estat: current EnvoyStats
        :return: the dictionary form of a DiagResult
        """

        result = DiagResult(self, estat, request)

        for group in self.ir.ordered_groups():
            # TCPMappings are currently handled elsewhere.
            if isinstance(group, IRHTTPMappingGroup):
                result.include_httpgroup(group)

        return result.as_dict()

    def lookup(self, request, key: str, estat: EnvoyStats) -> Optional[Dict[str, Any]]:
        """
        Generate data describing a specific key in the Ambassador setup, and all
        the things connected to it. Returns the dictionary form of a DiagResult.

        'key' can be a group key that starts with grp-, a cluster key that starts
        with cluster_, or a source key.

        :param request: the Flask request being handled
        :param key: the key of the thing we want
        :param estat: current EnvoyStats
        :return: the dictionary form of a DiagResult
        """

        result = DiagResult(self, estat, request)

        # Typically we'll get handed a group identifier here, but we might get
        # other stuff too, and we have to look for all of it.

        found: bool = False

        if key in self.groups:
            # Yup, group ID.
            group = self.groups[key]

            # TCPMappings are currently handled elsewhere.
            if isinstance(group, IRHTTPMappingGroup):
                result.include_httpgroup(group)

            found = True
        elif key in self.clusters:
            result.include_cluster(self.clusters[key].as_dict())
            found = True
        elif key in self.source_map:
            # The source_map is set up like:
            #
            # "mapping-qotm.yaml": {
            #     "mapping-qotm.yaml.1": true,
            #     "mapping-qotm.yaml.2": true,
            #     "mapping-qotm.yaml.3": true
            # }
            #
            # so for whatever we found, we need to tell the result to
            # include every element in the keys of the dict stored for
            # our key.
            for subkey in self.source_map[key].keys():
                result.include_element(subkey)
                # Not a typo. Set found here, in case somehow we land on
                # a key with no subkeys (which should be impossible, but,
                # y'know).
                found = True

        if found:
            result.finalize()
            return result.as_dict()
        else:
            return None
