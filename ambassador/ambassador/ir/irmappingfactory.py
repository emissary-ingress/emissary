from typing import Any, ClassVar, Dict, List, Optional, Tuple, Type, Union, TYPE_CHECKING

from ..config import Config

from .irbasemapping import IRBaseMapping
from .irhttpmapping import IRHTTPMapping
from .irtcpmapping import IRTCPMapping

if TYPE_CHECKING:
    from .ir import IR


class MappingFactory:
    @classmethod
    def load_all(cls, ir: 'IR', aconf: Config) -> None:
        cls.load_config(ir, aconf, "mappings", IRHTTPMapping)
        cls.load_config(ir, aconf, "tcpmappings", IRTCPMapping)

    @classmethod
    def load_config(cls, ir: 'IR', aconf: Config, config_name: str, mapping_class: Type[IRBaseMapping]) -> None:
        config_info = aconf.get_config(config_name)

        if not config_info:
            return

        assert(len(config_info) > 0)    # really rank paranoia on my part...

        for config in config_info.values():
            # ir.logger.debug("creating mapping for %s" % repr(config))

            mapping = mapping_class(ir, aconf, **config)
            ir.add_mapping(aconf, mapping)

    @classmethod
    def finalize(cls, ir: 'IR', aconf: Config) -> None:
        # OK. We've created whatever IRMappings we need. Time to create the clusters
        # they need.

        to_delete = []
        for group in ir.groups.values():
            group_id = group.group_id
            if not group.finalize(ir, aconf):
                to_delete.append(group_id)

        # delete any invalid mapping groups
        for group_id in to_delete:
            del ir.groups[group_id]
