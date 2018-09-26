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
from typing import cast as typecast

import sys

import json
import logging
import os
import re

from ..ir import IR


class DiagSource (dict):
    pass


class Diagnostics:
    ir: IR
    overview: Dict[str, dict]
    source_map: Dict[str, Dict[str, bool]]

    reKeyIndex = re.compile(r'\.(\d+)$')

    def __init__(self, ir: IR) -> None:
        self.ir = ir
        self.overview = self.generate_overview()

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
            'overview': self.overview,
            'source_map': self.source_map,
            'sources': self.sources,
            'groups': self.groups,
            'clusters': self.clusters
        }

    def generate_overview(self) -> Dict[str, Any]:
        self.logger = logging.getLogger("ambassador.diagnostics")
        self.logger.debug("---- generating overview")

        # Build a set of source _files_ rather than source _objects_.
        self.overview = {}
        self.source_map = {}
        self.sources = {}
        self.errors = {}

        for key, rsrc in self.ir.aconf.sources.items():
            key_base = key
            key_index = None

            if 'rkey' in rsrc:
                key_base, key_index = self.split_key(rsrc.rkey)

            location, _ = self.split_key(rsrc.get('location', key))

            self.logger.debug("  %s (%s, %s): %s @ %s" % (key, key_base, key_index, rsrc, location))

            src_map = self.source_map.setdefault(key_base, {})
            src_map[key] = True

            source_dict = self.sources.setdefault(
                location,
                {
                    'location': location,
                    'objects': {},
                    'count': 0,
                    'plural': "objects",
                    'error_count': 0,
                    'error_plural': "errors"
                }
            )

            source_key = key_base

            if key_index is not None:
                source_key += '.' + key_index

            raw_errors: List[Dict[str, str]] = self.ir.aconf.errors.get(source_key, [])
            errors = []

            for error in raw_errors:
                source_dict['error_count'] += 1

                errors.append({
                    'summary': error['error'].split('\n', 1)[0],
                    'text': error['error']
                })

            source_dict['error_plural'] = "error" if (source_dict['error_count'] == 1) else "errors"
            source_dict['count'] += 1
            source_dict['plural'] = "object" if (source_dict[ 'count' ] == 1) else "objects"

            object_dict = source_dict['objects']
            object_dict[source_key] = {
                'key': source_key,
                'kind': rsrc.kind,
                'errors': errors
            }

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
