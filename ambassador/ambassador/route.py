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
from .cluster import Cluster

#############################################################################
## route.py -- the route configuration object for Ambassador
##
## Envoy routing is a little weird. At the root of the world is an Envoy
## _listener_, which has a _filter chain_. Within the filter chain you will
## find (at least? exactly?) one HTTP connection manager filter. Within an
## HTTP connection manager you will find a _route configuration_, which
## contains (finally!) a set of _virtual hosts_ which form the route table.
##
## Route represents one of those virtual hosts within an Envoy routing
## table... sort of.
##
## A Route must have kind "Route" and location "-route-", and must have
## a Mapping's group_id as both rkey and name. 
##
## To find what sources are relevant for a given Route, look into things
## it's referenced_by.
##
## Route:
##


class VHost (Resource):
    """
    Virtual Host

    TODO: moar docstring.
    """

    def __init__(self, rkey: str, location: str="-vhost-", *,
                 name: str,
                 kind: str="Route",
                 apiVersion: Optional[str]=None,
                 serialization: Optional[str]=None):
        pass


class Route (Resource):
    """
    Routes are Resources with a bunch of extra stuff.

    TODO: moar docstring.
    """

    host_redirect: str
    prefix: str
    regex: Optional[str]
    shadow: bool
    weight: int
    precedence: int

    def __init__(self, rkey: str, location: str="-route-", *,
                 name: str,
                 kind: str="Route",
                 apiVersion: Optional[str]=None,
                 serialization: Optional[str]=None,

                 method: str='GET',
                 cluster: Cluster,
                 weight: int=100,
                 precedence: int=0,
                 prefix_rewrite: Optional[str]=None,
                 prefix_regex: bool=False,
                 host_redirect: bool=False,
                 shadow: bool=False,
                 headers: Optional[Dict]=None,
                 add_request_headers: Optional[Dict]=None,
                 cors: Optional[Dict]=None,
                 rate_limits: Optional[List[Dict]]=None,

                 **kwargs) -> None:
        """
        Initialize a Route from the raw fields of its Resource.
        """

        # First init our superclass...

        super().__init__(rkey, location,
                         kind=kind, name=name,
                         apiVersion=apiVersion,
                         serialization=serialization,
                         method=method,
                         cluster=cluster,
                         weight=weight,
                         precedence=precedence,
                         prefix_rewrite=prefix_rewrite,
                         prefix_regex=prefix_regex,
                         shadow=shadow,
                         headers=headers,
                         add_request_headers=add_request_headers,
                         cors=cors,
                         rate_limits=rate_limits,
                         **kwargs)

        # If prefix_regex is set, then the 'prefix' attribute is a regex, and needs to be
        # transformed into the 'regex' attribute. If not, the 'prefix' attribute is good
        # to go.
        #
        # XXX Is this really a good idea?

        if self.prefix_regex:
            self.regex = self.prefix
            del(self['prefix'])

        if not self.host_redirect and not self.shadow:
            self.clusters = [ { "name": self.cluster.name,
                                "weight": self.get("weight", None) } ]
        else:
            self.setdefault('clusters', [])

            if self.host_redirect and not self.shadow:
                self.host_redirect = self.service
            elif self.shadow:
                # If both shadow and host_redirect are set, we let shadow win.
                #
                # XXX CODE DUPLICATION with config.py!!
                # We're going to need to support shadow weighting later, so use a dict here.
                self.shadow = { 'name': self.cluster.name }
