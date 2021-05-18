#!/hint/python3
"""Generate terminal "control seqences" for ANSI X3.64 terminals.
Or rather, ECMA-48 terminals, because ANSI withdrew X3.64 in favor of
ECMA-48.
(https://www.ecma-international.org/publications/files/ECMA-ST/Ecma-048.pdf)
"control sequences" are a subset of "escape codes"; which are so named
because they start with the ASCII ESC character ("\033").
If you're going to try to read ECMA-48, be aware the notation they use
for a byte is "AB/CD" where AB and CD are zero-padded decimal numbers
that represent 4-bit sequences.  It's easiest to think of them as
hexadecimal byte; for example, when ECMA-48 says "11/15", think
"hexadecimal 0xBF".  And then to make sense of that hexadecimal
number, you'll want to have an ASCII reference table handy.
This implementation is not complete/exhaustive; it only supports the
things that we've found it handy to support.
"""

from typing import List, Union, cast, overload

_number = Union[int, float]


@overload
def cs(params: List[_number], op: str) -> str:
    ...


@overload
def cs(op: str) -> str:
    ...


def cs(arg1, arg2=None):  # type: ignore
    """cs returns a formatted 'control sequence' (ECMA-48 §5.4).
    This only supports text/ASCII ("7-bit") control seqences, and does
    support binary ("8-bit") control seqeneces.
    This only supports standard parameters (ECMA-48 §5.4.1.a /
    §5.4.2), and does NOT support "experimental"/"private" parameters
    (ECMA-48 §5.4.1.b).
    """
    csi = '\033['
    if arg2:
        params: List[_number] = arg1
        op: str = arg2
    else:
        params = []
        op = arg1
    return csi + (';'.join(str(n).replace('.', ':') for n in params)) + op


# The "EL" ("Erase in Line") control seqence (ECMA-48 §8.3.41) with no
# parameters.
clear_rest_of_line = cs('K')

# The "ED" ("Erase in Display^H^H^H^H^H^H^HPage") control seqence
# (ECMA-48 §8.3.39) with no parameters.
clear_rest_of_screen = cs('J')


def cursor_up(lines: int = 1) -> str:
    """Generate the "CUU" ("CUrsor Up") control sequence (ECMA-48 §8.3.22)."""
    if lines == 1:
        return cs('A')
    return cs([lines], 'A')


def _sgr_code(code: int) -> '_SGR':
    def get(self: '_SGR') -> '_SGR':
        return _SGR(self.params + [code])

    return cast('_SGR', property(get))


class _SGR:
    def __init__(self, params: List[_number] = []) -> None:
        self.params = params

    def __str__(self) -> str:
        return cs(self.params, 'm')

    reset = _sgr_code(0)
    bold = _sgr_code(1)
    fg_blk = _sgr_code(30)
    fg_red = _sgr_code(31)
    fg_grn = _sgr_code(32)
    fg_yel = _sgr_code(33)
    fg_blu = _sgr_code(34)
    fg_prp = _sgr_code(35)
    fg_cyn = _sgr_code(36)
    fg_wht = _sgr_code(37)
    # 38 is 8bit/24bit color
    fg_def = _sgr_code(39)


# sgr builds "Set Graphics Rendition" control sequences (ECMA-48
# §8.3.117).
sgr = _SGR()
