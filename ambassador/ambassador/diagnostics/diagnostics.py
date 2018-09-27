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
from ..envoy import EnvoyConfig
from .envoy_stats import EnvoyStats


class DiagSource (dict):
    pass


class DiagCluster (dict):
    def __init__(self, cluster) -> None:
        super().__init__(**cluster)

    def update_health(self, other: dict, keys: Optional[List[str]]=None) -> None:
        if not keys:
            keys = [ 'health', 'hmetric', 'hcolor' ]

        for key in keys:
            if key in other:
                dst_key = key

                if not dst_key.startswith("_"):
                    dst_key = "_%s" % dst_key

                self[dst_key] = other[key]

    @classmethod
    def unknown_cluster(cls):
        return DiagCluster({
            '_service': 'unknown service!',
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


class Diagnostics:
    ir: IR
    econf: EnvoyConfig
    estats: Optional[EnvoyStats]

    source_map: Dict[str, Dict[str, bool]]

    reKeyIndex = re.compile(r'\.(\d+)$')

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

            ambassador_element['objects'][fqkey] = {
                'key': fqkey,
                'kind': rsrc.kind,
                'errors': errors
            }

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

        self.groups = [ group.as_dict() for group in self.ir.groups.values()
                        if group.location != "--diagnostics--" ]

        self.clusters = [ cluster.as_dict() for cluster in self.ir.clusters.values()
                          if cluster.location != "--diagnostics--" ]

        # configuration = { key: self.envoy_config[key] for key in self.envoy_config.keys()
        #                   if key != "groups" }

        # cluster_to_service_mapping = {
        #     "cluster_ext_auth": "AuthService",
        #     "cluster_ext_tracing": "TracingService",
        #     "cluster_ext_ratelimit": "RateLimitService"
        # }
        #
        # ambassador_services = []
        #
        # for cluster in configuration.get('clusters', []):
        #     maps_to_service = cluster_to_service_mapping.get(cluster['name'])
        #     if maps_to_service:
        #         service_weigth = 100.0 / len(cluster['urls'])
        #         for url in cluster['urls']:
        #             ambassador_services.append(SourcedDict(
        #                 _from=cluster,
        #                 type=maps_to_service,
        #                 name=url,
        #                 cluster=cluster['name'],
        #                 _service_weight=service_weigth
        #             ))
        #
        # overview = dict(sources=sorted(source_files.values(), key=lambda x: x['filename']),
        #                 routes=groups,
        #                 **configuration)
        #
        # if len(ambassador_services) > 0:
        #     overview['ambassador_services'] = ambassador_services
        #
        # # self.logger.debug("overview result %s" % json.dumps(overview, indent=4, sort_keys=True))
        #
        # return overview

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
            'ambassador_elements': self.ambassador_elements,
            'envoy_elements': self.envoy_elements,
            'groups': self.groups,
            'clusters': self.clusters
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

    def route_and_cluster_info(self, request, cstats) -> List[Dict]:
        request_host = request.headers.get('Host', '*')
        request_scheme = request.headers.get('X-Forwarded-Proto', 'http').lower()

        cluster_info = DiagClusters(self.clusters)

        for cluster_name, cstat in cstats.items():
            cluster_info[cluster_name].update_health(cstat)

        route_info = []

        for group in self.groups:
            self.logger.debug("GROUP %s" % json.dumps(group, sort_keys=True, indent=4))

            prefix = group['prefix'] if 'prefix' in group else group['regex']
            rewrite = group.get('rewrite', "/")
            method = '*'
            host = None

            route_clusters: List[DiagCluster] = []
            route_cluster: Optional[DiagCluster] = None

            for mapping in group['mappings']:
                c_name = mapping['cluster']['name']

                route_cluster = DiagCluster({ 'weight': mapping['weight'] })
                route_cluster.update_health(cluster_info[c_name])

            if 'host_redirect' in group:
                    # XXX Stupid hackery here. redirect_cluster should be a real
                    # Cluster object.
                    redirect_cluster = {
                        '_service': group['host_redirect'],
                        'weight': 100,
                        'type_label': 'redirect'
                    }

                    route_cluster = DiagCluster(redirect_cluster)

                    self.logger.info("host_redirect route: %s" % group)
                    self.logger.info("host_redirect cluster: %s" % route_cluster)

            if 'shadow' in group:
                shadow_info = group['shadow']
                shadow_name = shadow_info.get('name', None)

                if shadow_name:
                    # XXX Stupid hackery here. shadow_cluster should be a real
                    # Cluster object.
                    shadow_cluster = {
                        '_service': shadow_name,
                        'weight': 100,
                        'type_label': 'shadow'
                    }

                    route_cluster = DiagCluster(shadow_cluster)

                    self.logger.info("shadow route: %s" % group)
                    self.logger.info("shadow cluster: %s" % route_cluster)

            route_clusters.append({
                'service': route_cluster.get('_service', 'unknown service!'),
                'weight': route_cluster.get('weight', 100),
                '_health': route_cluster.get('_hmetric', 'unknown'),
                '_hcolor': route_cluster.get('_hcolor', 'orange')
            })

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

            route_key = "%s://%s%s%s" % (request_scheme, host if host else request_host, sep, prefix)

            route_info.append({
                '_route': group,
                '_source': group['location'],
                '_group_id': group['group_id'],
                'key': route_key,
                'prefix': prefix,
                'rewrite': rewrite,
                'method': method,
                'headers': headers,
                'clusters': route_clusters,
                'host': host if host else '*'
            })

            self.logger.info("route_info")
            self.logger.info(json.dumps(route_info, indent=4, sort_keys=True))

            self.logger.info("cstats")
            self.logger.info(json.dumps(cstats, indent=4, sort_keys=True))

        return route_info

    def overview(self, request, estat: EnvoyStats) -> Dict[str, Any]:
        cluster_names = [ cluster['name'] for cluster in self.clusters ]
        cstats = { name: estat.cluster_stats(name) for name in cluster_names }

        route_info = self.route_and_cluster_info(request, cstats)

        return {
            'cstats': cstats,
            'route_info': route_info
        }
