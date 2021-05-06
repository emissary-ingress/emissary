#!/hint/python3

from . import ansiterm


def test_cursor_up() -> None:
    assert f'{ansiterm.cursor_up()}' == "\033[A"
    assert f'{ansiterm.cursor_up(1)}' == "\033[A"
    assert f'{ansiterm.cursor_up(3)}' == "\033[3A"


def test_sgr() -> None:
    assert f'{ansiterm.sgr.bold.fg_blu}' == "\033[1;34m"
