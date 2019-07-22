from typing import Any, ClassVar, Dict, List, Optional, Tuple, Union, TYPE_CHECKING

from ..config import Config

from .irresource import IRResource

if TYPE_CHECKING:
    from .ir import IR


class IRBaseMappingGroup (IRResource):
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

                # set mapping's new weight to current weight
                self.logger.debug(f"Assigning weight {current_weight} to mapping {mapping.name}")
                mapping.weight = current_weight

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

            self.logger.debug(f"Assigning weight {weight_per_weightless_mapping} of remaining weight {remaining_weight} to each of {num_weightless_mappings} weightless mappings")

            # Now, let's add weight to every weightless mapping and push to normalized_mappings
            for i, weightless_mapping in enumerate(weightless_mappings):

                # We need last mapping's weight to be 100
                if i == num_weightless_mappings - 1:
                    current_weight = 100
                else:
                    current_weight += weight_per_weightless_mapping

                self.logger.debug(f"Assigning weight {current_weight} to weightless mapping {weightless_mapping.name}")
                weightless_mapping['weight'] = current_weight
                normalized_mappings.append(weightless_mapping)

        self.mappings = normalized_mappings
        return True
