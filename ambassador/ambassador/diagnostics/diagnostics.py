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

from typing import Any, Dict, List, Optional, Tuple

import json
import logging
import re

from ..ir import IR
from ..ir.irmapping import IRMappingGroup
from ..envoy import EnvoyConfig
from .envoy_stats import EnvoyStats


class DiagSource (dict):
    pass


class DiagCluster (dict):
    def __init__(self, cluster) -> None:
        super().__init__(**cluster)

    def update_health(self, other: dict) -> None:
        for from_key, to_key in [
            ( 'health', '_health' ),
            ( 'hmetric', '_hmetric' ),
            ( 'hcolor', '_hcolor' )
        ]:
            if from_key in other:
                self[to_key] = other[from_key]

    def default_missing(self) -> dict:
        for key, default in [
            ('service', 'unknown service!'),
            ('weight', 100),
            ('_hmetric', 'unknown'),
            ('_hcolor', 'orange')
        ]:
            if not self.get(key, None):
                self[key] = default

        return dict(self)

    @classmethod
    def unknown_cluster(cls):
        return DiagCluster({
            'service': 'unknown service!',
            '_health': 'unknown cluster!',
            '_hmetric': 'unknown',
            '_hcolor': 'orange'
        })


class DiagClusters:
    def __init__(self, clusters: Optional[List[dict]] = []) -> None:
        self.clusters = {}

        for cluster in clusters:
            self[cluster['name']] = DiagCluster(cluster)

    def __getitem__(self, key: str) -> DiagCluster:
        if key not in self.clusters:
            self.clusters[key] = DiagCluster.unknown_cluster()

        return self.clusters[key]

    def __setitem__(self, key: str, value: DiagCluster) -> None:
        self.clusters[key] = value

    def __contains__(self, key: str) -> bool:
        return key in self.clusters

    def as_json(self):
        return json.dumps(self.clusters, sort_keys=True, indent=4)


class DiagResult:
    def __init__(self, diag: 'Diagnostics', estat: EnvoyStats, request) -> None:
        self.diag = diag
        self.logger = self.diag.logger
        self.estat = estat
        self.cluster_names = list(self.diag.clusters.keys())
        self.cstats = { name: self.estat.cluster_stats(name) for name in self.cluster_names }

        self.request_host = request.headers.get('Host', '*')
        self.request_scheme = request.headers.get('X-Forwarded-Proto', 'http').lower()

        self.clusters: Dict[str, DiagCluster] = {}
        self.routes: List[dict] = []
        self.element_keys: Dict[str, bool] = {}

        self.ambassador_resources = {}
        self.envoy_resources = {}

    def as_dict(self) -> Dict[str, Any]:
        return {
            'cluster_stats': self.cstats,
            'cluster_info': self.clusters,
            'route_info': self.routes,
            'ambassador_resources': self.ambassador_resources,
            'envoy_resources': self.envoy_resources
        }

    def include_element(self, key: str) -> None:
        self.element_keys[key] = True

    def include_referenced_elements(self, obj: dict) -> None:
        for element_key in obj['_referenced_by']:
            self.include_element(element_key)

    def include_cluster(self, cluster: dict) -> DiagCluster:
        c_name = cluster['name']

        if c_name not in self.clusters:
            self.clusters[c_name] = DiagCluster(cluster)

        if c_name in self.cstats:
            self.clusters[c_name].update_health(self.cstats[c_name])

        self.include_referenced_elements(cluster)

        return self.clusters[c_name]

    def include_group(self, group: IRMappingGroup) -> None:
        self.logger.debug("GROUP %s" % group.as_json())

        prefix = group['prefix'] if 'prefix' in group else group['regex']
        rewrite = group.get('rewrite', "/")
        method = '*'
        host = None

        route_cluster: Optional[DiagCluster] = None
        route_clusters: List[DiagCluster] = []

        for mapping in group.get('mappings', []):
            cluster = mapping['cluster']
            c_name = cluster['name']

            route_cluster = self.include_cluster(cluster.as_dict())
            route_cluster.update({'weight': mapping['weight']})

            self.logger.debug("GROUP %s CLUSTER %s %d%% (%s)" %
                              (group['group_id'], c_name, mapping['weight'], route_cluster))

            route_clusters.append(route_cluster)

        # Reset to pick up host_redirect or shadow.
        route_cluster = None

        host_redir = group.get('host_redirect', None)

        if host_redir:
            # XXX Stupid hackery here. redirect_cluster should be a real
            # Cluster object.
            redirect_cluster = self.include_cluster({
                'name': host_redir['name'],
                'service': host_redir['service'],
                'weight': 100,
                'type_label': 'redirect',
                '_referenced_by': [ host_redir['rkey'] ]
            })

            route_clusters.append(redirect_cluster)

            self.logger.info("host_redirect route: %s" % group)
            self.logger.info("host_redirect cluster: %s" % route_cluster)

        shadows = group.get('shadows', [])

        for shadow in shadows:
            shadow_cluster = self.include_cluster({
                'name': shadow['name'],
                'service': shadow['cluster']['name'],
                'weight': 100,
                'type_label': 'shadow',
                '_referenced_by': [ shadow['cluster'][ 'rkey' ] ]
            })

            route_clusters.append(shadow_cluster)

            self.logger.info("shadow route: %s" % group)
            self.logger.info("shadow cluster: %s" % route_cluster)

        headers = []

        for header in group.get('headers', []):
            hdr_name = header.get('name', None)
            hdr_value = header.get('value', None)

            if hdr_name == ':authority':
                host = hdr_value
            elif hdr_name == ':method':
                method = hdr_value
            else:
                headers.append(header)

        sep = "" if prefix.startswith("/") else "/"
        route_key = "%s://%s%s%s" % (self.request_scheme, host if host else self.request_host, sep, prefix)

        route_info = {
            '_route': group.as_dict(),
            '_source': group['location'],
            '_group_id': group['group_id'],
            'key': route_key,
            'prefix': prefix,
            'rewrite': rewrite,
            'method': method,
            'headers': headers,
            'clusters': [ x.default_missing() for x in route_clusters ],
            'host': host if host else '*'
        }

        self.routes.append(route_info)
        self.include_referenced_elements(group)

    def finalize(self) -> None:
        # Make sure the elements we've been messing with have sources and such.
        for key in self.element_keys.keys():
            amb_el_info = self.diag.ambassador_elements.get(key, None)

            if amb_el_info:
                # what about errors?

                for obj_key, obj in amb_el_info['objects'].items():
                    serialization = obj.get('serialization', None)

                    if serialization:
                        self.ambassador_resources[obj_key] = serialization

                    # What about errors?

            # Also make sure we have Envoy outputs for these things.
            envoy_el_info = self.diag.envoy_elements.get(key, None)

            if envoy_el_info:
                self.envoy_resources[key] = envoy_el_info

class Diagnostics:
    ir: IR
    econf: EnvoyConfig
    estats: Optional[EnvoyStats]

    source_map: Dict[str, Dict[str, bool]]

    reKeyIndex = re.compile(r'\.(\d+)$')

    filter_map = {
        'IRAuth': 'AuthService',
        'IRRateLimit': 'RateLimitService'
    }

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

        self.source_map = {}
        self.ambassador_elements = {}
        self.envoy_elements = {}

        self.errors = {}

        # First up, walk the list of Ambassador sources.
        for key, rsrc in self.ir.aconf.sources.items():
            uqkey = key
            fqkey = uqkey

            key_index = None

            if 'rkey' in rsrc:
                uqkey, key_index = self.split_key(rsrc.rkey)

            if key_index is not None:
                fqkey = "%s.%s" % (uqkey, key_index)

            location, _ = self.split_key(rsrc.get('location', key))

            self.logger.debug("  %s (%s): UQ %s, FQ %s, LOC %s" % (key, rsrc, uqkey, fqkey, location))

            self.remember_source(uqkey, fqkey, location, rsrc.rkey)

            ambassador_element = self.ambassador_elements.setdefault(
                fqkey,
                {
                    'location': location,
                    'objects': {},
                    'count': 0,
                    'plural': "objects",
                    'error_count': 0,
                    'error_plural': "errors"
                }
            )

            raw_errors: List[Dict[str, str]] = self.ir.aconf.errors.get(fqkey, [])
            errors = []

            for error in raw_errors:
                ambassador_element['error_count'] += 1

                errors.append({
                    'summary': error['error'].split('\n', 1)[0],
                    'text': error['error']
                })

            element = {
                'key': fqkey,
                'kind': rsrc.kind,
                'errors': errors,
            }

            serialization = rsrc.get('serialization', None)
            if serialization:
                element['serialization'] = serialization

            ambassador_element['objects'][fqkey] = element
            ambassador_element['error_plural'] = "error" if (ambassador_element['error_count'] == 1) else "errors"
            ambassador_element['count'] += 1
            ambassador_element['plural'] = "object" if (ambassador_element[ 'count' ] == 1) else "objects"

        # Next up, the Envoy elements.
        for kind, elements in self.econf.elements.items():
            for fqkey, envoy_element in elements.items():
                # The key here should already be fully qualified.
                uqkey, _ = self.split_key(fqkey)

                element_dict = self.envoy_elements.setdefault(fqkey, {})
                element_list = element_dict.setdefault(kind, [])
                element_list.append(envoy_element)

        self.groups = { 'grp-%s' % group.group_id: group for group in self.ir.groups.values()
                        if group.location != "--diagnostics--" }

        self.clusters = { cluster.name: cluster for cluster in self.ir.clusters.values()
                          if cluster.location != "--diagnostics--" }

        # configuration = { key: self.envoy_config[key] for key in self.envoy_config.keys()
        #                   if key != "groups" }

        self.ambassador_services = []

        for filter in self.ir.filters:
            self.logger.debug("FILTER %s" % filter.as_json())

            if filter.kind in Diagnostics.filter_map:
                type_name = Diagnostics.filter_map[filter.kind]
                self.add_ambassador_service(filter, type_name)

        if self.ir.tracing:
            self.add_ambassador_service(self.ir.tracing, 'TracingService (%s)' % self.ir.tracing.driver)

    def add_ambassador_service(self, svc, type_name) -> None:
        cluster = svc.cluster
        urls = cluster.urls

        svc_weight = 100.0 / len(urls)

        for url in urls:
            self.ambassador_services.append({
                'type': type_name,
                '_source': svc.location,
                'name': url,
                'cluster': cluster.name,
                '_service_weight': svc_weight
            })

    @staticmethod
    def split_key(key) -> Tuple[str, Optional[str]]:
        key_base = key
        key_index = None

        m = Diagnostics.reKeyIndex.search(key)

        if m:
            key_base = key[:m.start()]
            key_index = m.group(1)

        return key_base, key_index

    def as_dict(self) -> dict:
        return {
            'source_map': self.source_map,
            'ambassador_services': self.ambassador_services,
            'ambassador_elements': self.ambassador_elements,
            'envoy_elements': self.envoy_elements,
            'groups': { key: value.as_dict() for key, value in self.groups.items() },
            'clusters': { key: value.as_dict() for key, value in self.clusters.items() }
        }

    def _remember_source(self, src_key: str, dest_key: str):
        src_map = self.source_map.setdefault(src_key, {})
        src_map[dest_key] = True

    def remember_source(self, uqkey: str, fqkey: Optional[str], location: Optional[str], dest_key: str):
        self._remember_source(uqkey, dest_key)

        if fqkey and (fqkey != uqkey):
            self._remember_source(fqkey, dest_key)

        if location and (location != uqkey) and (location != fqkey):
            self._remember_source(location, dest_key)

    def overview(self, request, estat: EnvoyStats) -> Dict[str, Any]:
        result = DiagResult(self, estat, request)

        for group in self.ir.ordered_groups():
            result.include_group(group)

        return result.as_dict()

    def lookup(self, request, key: str, estat: EnvoyStats) -> Dict[str, Any]:
        result = DiagResult(self, estat, request)

        # Typically we'll get handed a group identifier here, but we might get
        # other stuff too.

        if key in self.groups:
            # Yup, group ID.
            group = self.groups[key]
            result.include_group(group)
            result.finalize()
            return result.as_dict()

        # elif key in self.source_map:
        #     element_keys = sorted(self.source_map[key].keys())
        #
        #
        #     return result.as_dict()
        else:
            return None

