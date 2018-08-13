from typing import ClassVar, Optional

from ..config import Config
from ..resource import Resource


class IRResource (Resource):
    """
    A resource within the IR.
    """

    modules_handled: ClassVar[Optional[str]] = None

    _active: bool = False

    def __init__(self, ir: 'IR', aconf: Config, rkey: str, kind: str, name: str,
                 **kwargs) -> None:
        # print("IRResource __init__ (%s %s)" % (kind, name))

        super().__init__(rkey, "-ir-",
                         kind=kind, name=name,
                         apiVersion="ambassador/ir",
                         **kwargs)

        self.logger = ir.logger

        if self.setup(ir, aconf):
            self.set_active(True)

    def set_active(self, active: bool) -> None:
        self._active = active

    def is_active(self) -> bool:
        return self._active

    def __nonzero__(self) -> bool:
        return self._active and not self._errors

    def setup(self, ir: 'IR', aconf: Config) -> bool:
        # If you don't override setup, you end up with an IRResource that's always active.
        return True
