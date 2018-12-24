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

import os

from typing import ClassVar, TYPE_CHECKING

from ..config import Config
from .irresource import IRResource as IRResource
from ambassador.utils import RichStatus

if TYPE_CHECKING:
    from .ir import IR


#############################################################################
## tls.py -- the tls_context configuration object for Ambassador
##
## IRAmbassadorTLS represents an Ambassador TLS configuration: it's the way
## we unify the TLS module and the 'tls' block in the Ambassador module. This
## class is pretty much all about managing priority between the two -- any
## important information here gets turned into IRTLSContext objects before
## TLS configuration actually happens.
##
## There's a fair amount of logic around making priority decisions between
## the 'tls' block and the TLS module at present. Probably that logic should
## migrate here, or this class should go away.


class IRAmbassadorTLS (IRResource):
    def __init__(self, ir: 'IR', aconf: Config,
                 rkey: str="ir.tlsmodule",
                 kind: str="IRTLSModule",
                 name: str="ir.tlsmodule",
                 enabled: bool=True,

                 **kwargs) -> None:
        """
        Initialize an IRAmbassadorTLS from the raw fields of its Resource.
        """

        # ir.logger.debug("IRAmbassadorTLS __init__ (%s %s %s)" % (kind, name, kwargs))

        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, kind=kind, name=name,
            enabled=enabled,
            **kwargs
        )


class TLSModuleFactory:
    @classmethod
    def load_all(cls, ir: 'IR', aconf: Config) -> None:
        assert ir

        tls_module = aconf.get_module('tls')

        if tls_module:
            # ir.logger.debug("TLSModuleFactory saving TLS module: %s" % tls_module.as_json())

            # XXX What a hack. IRAmbassadorTLS.from_resource() should be able to make
            # this painless.
            new_args = dict(tls_module.as_dict())
            new_rkey = new_args.pop('rkey', tls_module.rkey)
            new_kind = new_args.pop('kind', tls_module.kind)
            new_name = new_args.pop('name', tls_module.name)
            new_location = new_args.pop('location', tls_module.location)

            ir.tls_module = IRAmbassadorTLS(ir, aconf,
                                            rkey=new_rkey,
                                            kind=new_kind,
                                            name=new_name,
                                            location=new_location,
                                            **new_args)

            # ir.logger.debug("TLSModuleFactory saved TLS module: %s" % ir.tls_module.as_json())

    @classmethod
    def finalize(cls, ir: 'IR', aconf: Config) -> None:
        pass
