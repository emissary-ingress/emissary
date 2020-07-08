from typing import ContextManager, List, Optional

import contextlib
import dataclasses


@dataclasses.dataclass
class Location:
    """
    Represents a location for parsing.
    """

    filename: Optional[str] = None
    ocount: int = 1

    def mark_annotation(self) -> None:
        if self.filename is None:
            return
        elif self.filename.endswith(':annotation'):
            return

        self.filename += ':annotation'

    def __str__(self) -> str:
        return f"{self.filename or 'anonymous YAML'}.{self.ocount}"


class LocationManager:
    """
    Manages locations contextually.
    """

    previous: List[Location]
    current: Location

    def __init__(self) -> None:
        self.previous = []
        self.current = Location()

    def push(self, filename: Optional[str] = None, ocount: int = 1) -> ContextManager[Location]:
        current = Location(filename, ocount)
        self.previous.append(self.current)
        self.current = current

        # This trick lets you use the return value of this method in a `with`
        # statement. At the conclusion of the statement block, the location will
        # automatically be popped from the stack.
        @contextlib.contextmanager
        def popper():
            yield current
            self.pop()

        return popper()

    def push_reset(self) -> ContextManager[Location]:
        """
        Like push, but simply resets ocount keeping the current filename. Useful
        for changing resource types.
        """
        return self.push(filename=self.current.filename)

    def pop(self) -> Location:
        current = self.current
        self.current = self.previous.pop()
        return current
