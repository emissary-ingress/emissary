#!/hint/python3

import contextlib
import io
import os

from . import ansiterm, uiutil


def test_indent_basic() -> None:
    buff = io.StringIO()
    indent = uiutil.Indent(output=buff, indent='>')
    indent.write("foo")
    assert buff.getvalue() == ">foo"
    indent.write("\nbar\n")
    assert buff.getvalue() == ">foo\n>bar\n"
    indent.write("qux")
    assert buff.getvalue() == ">foo\n>bar\n>qux"
    indent.write("\r")
    assert buff.getvalue() == ">foo\n>bar\n>qux\r"
    indent.write("wombat")
    assert buff.getvalue() == ">foo\n>bar\n>qux\r>wombat"


def test_indent_wrap_1() -> None:
    buff = io.StringIO()
    stdio = uiutil.Indent(output=buff, indent='>', columns=5)

    stdio.write("abcdef")
    assert buff.getvalue() == ">abcd\n>ef"


def test_indent_wrap_2() -> None:
    buff = io.StringIO()
    stdio = uiutil.Indent(output=buff, indent='>', columns=5)

    stdio.write("abcd")
    assert buff.getvalue() == ">abcd"


def test_indent_1() -> None:
    buff = io.StringIO()
    stdio = uiutil.Indent(output=buff, indent='   > ', columns=80)

    stdio.write(f"this is a line{ansiterm.clear_rest_of_screen}\nnextline")
    assert buff.getvalue() == f"   > this is a line{ansiterm.clear_rest_of_screen}\n   > nextline"


def test_check_wrap() -> None:
    # Given columns=20
    #   "00000000001111111112"
    #   "12345678901234567890"
    # And the unwrapped input
    #   " [stat] mytest\n"
    #   "   > this is a long line\n"
    # We should get that as
    #   " [stat] mytest\n"
    #   "   > this is a long \n"
    #   "   > line\n"
    #
    # So we test that it correctly detects that wrap, and knows that
    # it has to move the cursor 3 lines instead of 2 when filling in
    # the final status.

    os.environ["COLUMNS"] = "80"
    buff = io.StringIO()
    with contextlib.redirect_stdout(buff):
        checker = uiutil.Checker()
        with checker.check("mytest", clear_on_success=False):
            print("this is a long line")
    assert buff.getvalue() == "".join([
        " \033[1;34m[....]\033[m mytest\n",  # initial output.write()
        "   > this is a long line\n",  # yield check
        "\r\033[2A",  # output.goto_line(1)
        " \033[1;32m[ OK ]\033[m mytest\033[K",  # final output.write()
        "\r\n\n",  # output.goto_line(end)
    ])

    os.environ["COLUMNS"] = "20"
    buff = io.StringIO()
    with contextlib.redirect_stdout(buff):
        checker = uiutil.Checker()
        with checker.check("mytest", clear_on_success=False):
            print("this is a long line")
    assert buff.getvalue() == "".join([
        " \033[1;34m[....]\033[m mytest\n",  # initial output.write()
        "   > this is a long \n",  # yield check
        "   > line\n",
        "\r\033[3A",  # output.goto_line(1)
        " \033[1;32m[ OK ]\033[m mytest\033[K",  # final output.write()
        "\r\n\n\n",  # output.goto_line(end)
    ])
