from typing import Any, ClassVar, Dict, List, Optional, Tuple, Union, TYPE_CHECKING

from ..config import Config

from .irresource import IRResource
from .irbasemapping import IRBaseMapping

if TYPE_CHECKING:
    from .ir import IR # pragma: no cover


class IRBaseMappingGroup (IRResource):
    mappings: List[IRBaseMapping]
    group_id: str
    group_weight: List[Union[str, int]]
    labels: Dict[str, Any]
    _cache_key: Optional[str]

    def __init__(self, ir: 'IR', aconf: Config,
                 location: str,
                 rkey: str="ir.mappinggroup",
                 kind: str="IRBaseMappingGroup",
                 name: str="ir.mappinggroup",
                 **kwargs) -> None:
        # Default to no cache key...
        self._cache_key = None

        # Default to no mappings...
        self.mappings = []

        # ...before we init the superclass, which will call self.setup().
        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, location=location,
            kind=kind, name=name, **kwargs
        )

    @classmethod
    def key_for_id(cls, group_id: str) -> str:
        return f"{cls.__name__}-{group_id}"

    # XXX WTFO, I hear you cry. Why is this "type: ignore here?" So here's the deal:
    # mypy doesn't like it if you override just the getter of a property that has a
    # setter, too, and I cannot figure out how else to shut it up.
    @property   # type: ignore
    def cache_key(self) -> str:
        # XXX WTFO, I hear you cry again! Can this possibly be thread-safe??!
        # Well, no, not really. But as long as you're not trying to use the
        # cache_key before actually initializing this group, key_for_id()
        # will be idempotent, so it doesn't matter.

        if not self._cache_key:
            self._cache_key = self.__class__.key_for_id(self.group_id)

        return self._cache_key

    def normalize_weights_in_mappings(self):
        weightless_mappings = []
        num_weightless_mappings = 0

        normalized_mappings = []

        current_weight = 0
        for mapping in self.mappings:
            if 'weight' in mapping:
                if mapping.weight > 100:
                    self.post_error(f"Mapping {mapping.name} has invalid weight {mapping.weight}")
                    return False

                # increment current weight by mapping's weight
                current_weight += round(mapping.weight)

                # set mapping's calculated weight to current weight
                self.logger.debug(f"Assigning calculated weight {current_weight} to mapping {mapping.name}")
                mapping._weight = current_weight

                # add this mapping to normalized mappings
                normalized_mappings.append(mapping)
            else:
                num_weightless_mappings += 1
                weightless_mappings.append(mapping)

        if current_weight > 100:
            self.post_error(f"Total weight of mappings exceed 100, please reconfigure for correct behavior...")
            return False

        if num_weightless_mappings > 0:
            remaining_weight = 100 - current_weight
            weight_per_weightless_mapping = round(remaining_weight/num_weightless_mappings)

            self.logger.debug(f"Assigning calculated weight {weight_per_weightless_mapping} of remaining weight {remaining_weight} to each of {num_weightless_mappings} weightless mappings")

            # Now, let's add weight to every weightless mapping and push to normalized_mappings
            for i, weightless_mapping in enumerate(weightless_mappings):

                # We need last mapping's weight to be 100
                if i == num_weightless_mappings - 1:
                    current_weight = 100
                else:
                    current_weight += weight_per_weightless_mapping

                self.logger.debug(f"Assigning weight {current_weight} to weightless mapping {weightless_mapping.name}")
                weightless_mapping._weight = current_weight
                normalized_mappings.append(weightless_mapping)

        self.mappings = normalized_mappings
        return True
