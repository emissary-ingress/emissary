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

from typing import Optional

from .acresource import ACResource

#############################################################################
## mapping.py -- the mapping configuration object for Ambassador
##
## Each Mapping object has a group_id that reflects the group of Mappings
## that it is a part of. By definition, two Mappings with the same group_id
## are reflecting a single mapped resource that's going to multiple services.
## This implies that Mapping.group_id() is a very, very, very important
## thing that can have dramatic customer impact if changed! (At some point,
## we should probably allow the human writing the Mapping to override the
## grouping, in much the same way we allow overriding precedence.)
##
## Each Mapping object also has a weight, which is used for ordering. The
## default is computing in Mapping.route_weight(), but it can be overridden
## using the precedence field in the Mapping object.

# StringOrBool = Union[str, bool]
# DictOfStringOrBool = Dict[str, StringOrBool]


class ACMapping(ACResource):
    """
    ACMappings are ACResources with a bunch of extra stuff.

    TODO: moar docstring.
    """

    def __init__(
        self,
        rkey: str,
        location: str,
        *,
        name: str,
        kind: str = "Mapping",
        apiVersion: Optional[str] = None,
        serialization: Optional[str] = None,
        service: str,
        prefix: str,
        prefix_regex: bool = False,
        rewrite: Optional[str] = "/",
        case_sensitive: bool = False,
        grpc: bool = False,
        bypass_auth: bool = False,
        bypass_error_response_overrides: bool = False,
        # We don't list "method" or "method_regex" above because if they're
        # not present, we want them to be _not present_. Having them be always
        # present with an optional method is too annoying for schema validation
        # at this point.
        **kwargs
    ) -> None:
        """
        Initialize an ACMapping from the raw fields of its ACResource.
        """

        # print("ACMapping __init__ (%s %s)" % (kind, name))

        # First init our superclass...

        super().__init__(
            rkey,
            location,
            kind=kind,
            name=name,
            apiVersion=apiVersion,
            serialization=serialization,
            service=service,
            prefix=prefix,
            prefix_regex=prefix_regex,
            rewrite=rewrite,
            case_sensitive=case_sensitive,
            grpc=grpc,
            bypass_auth=bypass_auth,
            bypass_error_response_overrides=bypass_error_response_overrides,
            **kwargs
        )
