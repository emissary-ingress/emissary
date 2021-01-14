from typing import Dict, List, Optional, TYPE_CHECKING

from ..config import Config
from ..utils import dump_json

from .irresource import IRResource

if TYPE_CHECKING:
    from .ir import IR


class IRCircuitBreakers(IRResource):
    """
    IRCircuitBreaker is an IRResource for handling circuit breakers in Envoy clusters
    """

    circuit_breakers: (list, tuple)

    def __init__(self, ir: 'IR', aconf: Config,
                 rkey: str="ir.circuitbreaker",
                 name: str="ir.circuitbreaker",
                 kind: str="IRCircuitBreaker",
                 circuit_breakers: (list, tuple)=None,
                 **kwargs) -> None:
        """
        Initialize IRCircuitBreakers
        """

        super().__init__(
            ir=ir, aconf=aconf, rkey=rkey, kind=kind, name=name,
            circuit_breakers=circuit_breakers, **kwargs)

    def setup(self, ir: 'IR', aconf: Config) -> bool:
        """
        Set up an IRCircuitBreakers based on the circuit breakers passed into
        __init__.
        """

        circuit_breakers: Optional[List[Dict[str, Dict]]] = self.get("circuit_breakers", None)

        assert circuit_breakers is not None

        ir.logger.debug(f"CIRCUIT BREAKERS: {circuit_breakers}")

        if not self.validate_circuit_breakers(ir, circuit_breakers):
            self.post_error("Invalid circuit_breakers specified: {}, invalidating mapping".format(circuit_breakers))

    @staticmethod
    def validate_circuit_breakers(ir: 'IR', circuit_breakers) -> bool:
        if not isinstance(circuit_breakers, (list, tuple)):
            return False

        for circuit_breaker in circuit_breakers:
            if '_name' in circuit_breaker:
                # Already reconciled.
                ir.logger.debug(f'Breaker validation: good breaker {circuit_breaker["_name"]}')
                continue

            ir.logger.debug(f'Breaker validation: {dump_json(circuit_breakers, pretty=True)}')

            name_fields = [ 'cb' ]

            if 'priority' in circuit_breaker:
                prio = circuit_breaker.get('priority').lower()
                if prio not in ['default', 'high']:
                    return False

                name_fields.append(prio[0])
            else:
                name_fields.append('n')

            digit_fields = [ ( 'max_connections', 'c' ),
                             ( 'max_pending_requests', 'p' ),
                             ( 'max_requests', 'r' ),
                             ( 'max_retries', 't' ) ]

            for field, abbrev in digit_fields:
                if field in circuit_breaker:
                    try:
                        value = int(circuit_breaker[field])
                        name_fields.append(f'{abbrev}{value}')
                    except ValueError:
                        return False

            circuit_breaker['_name'] = ''.join(name_fields)
            ir.logger.debug(f'Breaker valid: {circuit_breaker["_name"]}')

        return True

    def as_dict(self) -> dict:
        return self.circuit_breakers
