from typing import Any, ClassVar, Dict, List, Optional, Tuple, Type, Union, TYPE_CHECKING

from ..config import Config

from .irbasemapping import IRBaseMapping
from .irhttpmapping import IRHTTPMapping
from .irtcpmapping import IRTCPMapping

if TYPE_CHECKING:
    from .ir import IR # pragma: no cover


def unique_mapping_name(aconf: Config, name: str) -> str:
    http_mappings = aconf.get_config('mappings') or {}
    tcp_mappings = aconf.get_config('tcpmappings') or {}

    basename = name
    counter = 0

    while name in http_mappings or name in tcp_mappings:
        name = f"{basename}-{counter}"
        counter += 1

    return name


class MappingFactory:
    @classmethod
    def load_all(cls, ir: 'IR', aconf: Config) -> None:
        cls.load_config(ir, aconf, "Mapping", "mappings", IRHTTPMapping)
        cls.load_config(ir, aconf, "TCPMapping", "tcpmappings", IRTCPMapping)

    @classmethod
    def load_config(cls, ir: 'IR', aconf: Config,
                    kind: str, config_name: str, mapping_class: Type[IRBaseMapping]) -> None:
        config_info = aconf.get_config(config_name)

        if not config_info:
            return

        assert(len(config_info) > 0)    # really rank paranoia on my part...

        for config in config_info.values():
            # ir.logger.debug("creating mapping for %s" % repr(config))

            # Is this mapping already in the cache?
            key = IRBaseMapping.make_cache_key(kind, config.name, config.namespace)

            mapping: Optional[IRBaseMapping] = None
            cached_mapping = ir.cache_fetch(key)

            if cached_mapping is None:
                # Cache miss: synthesize a new Mapping.
                ir.logger.debug(f"IR: synthesizing Mapping for {config.name}")
                mapping = mapping_class(ir, aconf, **config)
            else:
                # Cache hit. We know a priori that anything in the cache under a Mapping
                # key must be an IRBaseMapping, but let's assert that rather than casting.
                assert(isinstance(cached_mapping, IRBaseMapping))
                mapping = cached_mapping

            ir.logger.debug(f"IR: adding Mapping for {config.name}")
            ir.add_mapping(aconf, mapping)

        ir.cache.dump("MappingFactory")

    @classmethod
    def finalize(cls, ir: 'IR', aconf: Config) -> None:
        # OK. We've created whatever IRMappings we need. Time to create the clusters
        # they need.

        ir.logger.debug("IR: MappingFactory finalizing")

        for group in ir.groups.values():
            ir.logger.debug("IR: MappingFactory finalizing group %s", group.group_id)
            group.finalize(ir, aconf)
            ir.logger.debug("IR: MappingFactory finalized group %s", group.group_id)

        ir.logger.debug("IR: MappingFactory finalized")
