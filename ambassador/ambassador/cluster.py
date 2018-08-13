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

from typing import Dict, List, Optional
from typing import cast as typecast

from .resource import Resource
from .ir.tls import TLSContext

#############################################################################
## cluster.py -- the cluster configuration object for Ambassador
##
## Cluster represents an Envoy cluster: a collection of endpoints that
## provide a single service. Clusters get used for quite a few different
## things in Ambassador -- they are basically the generic "upstream service"
## entity.
##
## A Cluster must have kind "Cluster" and location "-cluster-", and will
## always have identical rkey and name. This name is used to identify
## the cluster within Envoy.
##
## To find what sources are relevant for a given Cluster, look into things
## it's referenced_by.


class Cluster (Resource):
    """
    Clusters are Resources with a bunch of extra stuff.

    TODO: moar docstring.
    """

    def __init__(self, res_key: str, location: str="-cluster-", *,
                 name: str,
                 kind: str="Cluster",
                 apiVersion: str=None,
                 serialization: Optional[str]=None,

                 discovery: str="strict_dns",
                 lb_type: str="round_robin",
                 endpoints: List[str],
                 connect_timeout_ms: int=3000,
                 grpc: bool=False,

                 tls_context: Optional[TLSContext]=None,
                 breakers: Optional[List[Resource]]=None,
                 outliers: Optional[List[Resource]]=None,
                 max_requests_per_connection: Optional[int]=None,
                 features: Optional[List[str]]=None,
                 http2_settings: Optional[Dict[str, str]]=None,

                 **kwargs) -> None:
        """
        Initialize a Cluster from the raw fields of its Resource.
        """

        # First init our superclass...

        super().__init__(res_key, location,
                         name=name,
                         kind=kind,
                         apiVersion=apiVersion,
                         serialization=serialization,
                         discovery=discovery,
                         lb_type=lb_type,
                         endpoints=endpoints,
                         connect_timeout_ms=connect_timeout_ms,
                         tls_context=tls_context,
                         breakers=breakers,
                         outliers=outliers,
                         max_requests_per_connection=max_requests_per_connection,
                         features=features,
                         http2_settings=http2_settings,
                         **kwargs)

        self.tls_array: Optional[List[Dict[str, str]]] = None

        if self.tls_context:
            self.references(self.tls_context)
            tls_array: List[Dict[str, str]] = []

            for key, value in self.tls_context.items():
                if key.startswith('_'):
                    continue

                tls_array.append({'key': key, 'value': value})

            if self.host_rewrite:
                tls_array.append({ 'key': 'sni',
                                   'value': typecast(str, self.host_rewrite) })

            self.tls_array = sorted(tls_array, key=lambda x: x['key'])

        if self.grpc:
            self.features = self.setdefault(features, [])

            if 'http2' not in self.features:
                self.features.append('http2')
