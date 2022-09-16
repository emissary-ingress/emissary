#!/hint/python3

import io
import os
import re
import shutil
import string
import subprocess
import sys
from contextlib import contextmanager
from traceback import print_exc
from typing import Callable, Generator, List, Optional, TextIO, Tuple

from . import ansiterm

# run()/run_bincapture()/run_txtcapture() and capture_output() seem like they're
# re-implementing something that should already exist for us to use.  And
# indeed, the `run*()` functions are essentially just stdlib `subprocess.run()`
# and `capture_output()` is essentially just stdlib
# `contextlib.redirect_stdout()`+`contextlib.redirect_stderr()`.  But the big
# reason for them to exist here is: `contextlib.redirect_*` and `subprocess`
# don't play together!  It's infuriating.
#
# So we define a global `_capturing` that is set while `capture_output()` is
# running, and have the `run*()` functions adjust their behavior if it's set.
# We could more generally do this by wrapping either `subprocess.run()` or
# `contextlib.redirect_*()`.  If we only ever called the redirect/capture
# function on a real file with a real file descriptor, it would be hairy, but
# not _too_ hairy[1].  But we want to call the redirect/capture function with
# not-a-real-file things like Indent or LineTracker.  So we'd have to get even
# hairier... we'd have to do a bunch of extra stuff when the output's
# `.fileno()` raises io.UnsupportedOperation; the same way that Go's
# `os/exec.Cmd` has to do extra stuff when the output ins't an `*os.File` (and
# that's one of the big reasons why I've said that Go's "os/exec" is superior to
# other languages subprocess facilities).  And it's my best judgment that just
# special-casing it with `_capturing` is the better choice than taking on all
# the complexity of mimicing Go's brilliance.
#
# [1]: https://eli.thegreenplace.net/2015/redirecting-all-kinds-of-stdout-in-python/

_capturing = False


def check_command(args) -> bool:
    p = subprocess.run(args, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    return p.returncode == 0


def run(args: List[str]) -> None:
    """run is like "subprocess.run(args)", but with helpful settings and
    obeys "with capture_output(out)".
    """
    if _capturing:
        try:
            subprocess.run(args, check=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
        except subprocess.CalledProcessError as err:
            raise Exception(f"{err.stdout}{err}") from err
    else:
        subprocess.run(args, check=True)


def run_bincapture(args: List[str]) -> bytes:
    """run is like "subprocess.run(args, capture_out=True, text=False)",
    but with helpful settings and obeys "with capture_output(out)".
    """
    if _capturing:
        try:
            return subprocess.run(args, check=True, capture_output=True).stdout
        except subprocess.CalledProcessError as err:
            raise Exception(f"{err.stderr.decode('UTF-8')}{err}") from err
    else:
        return subprocess.run(args, check=True, stdout=subprocess.PIPE).stdout


def run_txtcapture(args: List[str]) -> str:
    """run is like "subprocess.run(args, capture_out=True, text=true)",
    but with helpful settings and obeys "with capture_output(out)".
    """
    if _capturing:
        try:
            out = subprocess.run(args, check=True, capture_output=True, text=True).stdout
        except subprocess.CalledProcessError as err:
            raise Exception(f"{err.stderr}{err}") from err
    else:
        out = subprocess.run(args, check=True, stdout=subprocess.PIPE, text=True).stdout
    if out.endswith("\n"):
        out = out[:-1]
    return out


@contextmanager
def capture_output(log: io.StringIO) -> Generator[None, None, None]:
    """capture_output is like contextlib.redirect_stdout but also
    redirects stderr, and also does some extra stuff so that we can
    have run/run_bincapture/run_txtcapture functions that obey it.
    """
    global _capturing

    saved_capturing = _capturing
    saved_stdout = sys.stdout
    saved_stderr = sys.stderr

    _capturing = True
    sys.stdout = sys.stderr = log
    try:
        yield
    finally:
        _capturing = saved_capturing
        sys.stdout = saved_stdout
        sys.stderr = saved_stderr


def _lex_char_or_cs(text: str) -> Tuple[str, str]:
    """Look atthe beginning of the given text and trim either a byte, or
    an ANSI control sequence from the beginning, returning a tuple
    ("char-or-cs", "remaining-text").  If it looks like the text is a
    truncated control seqence, then it doesn't trim anything, and
    returns ("", "original"); signaling that it needs to wait for more
    input before successfully lexing anything.
    """
    if text == '\033':
        # wait to see if this is a control sequence
        return '', text
    i = 1
    if text.startswith('\033['):
        try:
            i = len('\033[')
            while text[i] not in string.ascii_letters:
                i += 1
            i += 1
        except IndexError:
            # wait for a complete control sequence
            return '', text
    return text[:i], text[i:]


class Indent(io.StringIO):
    """Indent() is like a io.StringIO(), will indent text with the given
    string.
    """
    def __init__(self, indent: str = "", output: Optional[TextIO] = None, columns: Optional[int] = None) -> None:
        """Arguments:
          indent: str: The string to indent with.
          output: Optional[TextIO]: A TextIO to write to, instead of
                  building an in-memory buffer.
          columns: Optional[int]: How wide the terminal is; this is
                   imporant because a line wrap needs to trigger an
                   indent.  If not given, then 'output.columns' is
                   used if 'output' is set and has a 'columns'
                   attribute, otherwise shutil.get_terminal_size() is
                   used.  Use a value <= 0 to explicitly disable
                   wrapping.
        The 'columns' attribute on the resulting object is set to the
        number of usable colums; "arg_columns - len(indent)".  This
        allows Indent objects to be nested.
        Indent understands "\r" and "\n", but not "\t" or ANSI control
        sequences that move the cursor; it assumes that all ANSI
        control sequences do not move the cursor.
        """
        super().__init__()
        self._indent = indent
        self._output = output

        if columns is None:
            if output and hasattr(output, 'columns'):
                columns = output.columns  # type: ignore
            else:
                columns = shutil.get_terminal_size().columns
        self.columns = columns - len(self._indent)

    _rest = ""
    _cur_col = 0
    # 0: no indent has been printed for this line, and indent will need to be printed unless this is the final trailing NL
    # 1: an indent needs to be printed for this line IFF there is any more output on it
    # 2: no indent (currently) needs to be printed for this line
    _print_indent = 0

    def write(self, text: str) -> int:
        # This algorithm is based on
        # https://git.parabola.nu/packages/libretools.git/tree/src/chroot-tools/indent
        self._rest += text

        while self._rest:
            c, self._rest = _lex_char_or_cs(self._rest)
            if c == "":
                # wait for more input
                break
            elif c == "\n":
                if self._print_indent < 1:
                    self._write(self._indent)
                self._write(c)
                self._print_indent = 0
                self._cur_col = 0
            elif c == "\r":
                self._write(c)
                self._print_indent = min(self._print_indent, 1)
                self._cur_col = 0
            elif c.startswith('\033['):
                if self._print_indent < 2:
                    self._write(self._indent)
                self._write(c)
                self._print_indent = 2
            elif self.columns > 0 and self._cur_col >= self.columns:
                self._rest = "\n" + c + self._rest
            else:
                if self._print_indent < 2:
                    self._write(self._indent)
                self._write(c)
                self._print_indent = 2
                self._cur_col += len(c)
        return len(text)

    def _write(self, text: str) -> None:
        if self._output:
            self._output.write(text)
        else:
            super().write(text)

    def flush(self) -> None:
        if self._output:
            self._output.flush()
        else:
            super().flush()

    def input(self) -> str:
        """Use "myindent.input()" instead of "input()" in order to nest well
        with LineTrackers.
        """
        if hasattr(self._output, 'input'):
            text: str = self._output.input()  # type: ignore
        else:
            text = input()
        return text


class LineTracker(io.StringIO):
    """LineTracker() is like a io.StringIO(), but will keep track of which
    line you're on; starting on line "1".
    LineTracker understands "\n", and the "cursor-up" (CSI-A) control
    sequence.  It does not detect wrapped lines; use Indent() to turn
    those in to hard-wraps that LineTracker understands.
    """
    def __init__(self, output: Optional[TextIO] = None) -> None:
        self._output = output
        if output and hasattr(output, 'columns'):
            self.columns = output.columns  # type: ignore

    cur_line = 1

    _rest = ""

    def _handle(self, text: str) -> None:
        self._rest += text
        while self._rest:
            c, self._rest = _lex_char_or_cs(self._rest)
            if c == "":
                # wait for more input
                break
            elif c == "\n":
                self.cur_line += 1
            elif c.startswith("\033[") and c.endswith('A'):
                lines = int(c[len("\033["):-len('A')] or "1")
                self.cur_line -= lines

    def input(self) -> str:
        """Use "mylinetracker.input()" instead of "input()" to avoid the
        LineTracker not seeing any newlines input by the user.
        """
        if hasattr(self._output, 'input'):
            text: str = self._output.input()  # type: ignore
        else:
            text = input()
        self._handle(text + "\n")
        return text

    def goto_line(self, line: int) -> None:
        """goto_line moves the cursor to the beginning of the given line;
        where line 1 is the line that the LineTracker started on, line
        0 is the line above that, and line 1 is the line below
        that.
        """
        self.write("\r")
        if line < self.cur_line:
            total_lines = shutil.get_terminal_size().lines
            if (self.cur_line - line) >= total_lines:
                raise Exception(f"cannot go back {self.cur_line - line} lines (limit={total_lines - 1})")
            self.write(ansiterm.cursor_up(self.cur_line - line))
        else:
            self.write("\n" * (line - self.cur_line))

    def write(self, text: str) -> int:
        self._handle(text)
        if self._output:
            return self._output.write(text)
        else:
            return super().write(text)

    def flush(self) -> None:
        if self._output:
            self._output.flush()
        else:
            super().flush()


class Checker:
    """Checker is a terminal UI widget for printing a series of '[....]'
    (running) / '[ OK ]' / '[FAIL]' checks where we can diagnostic
    output while the check is running, and then go back and update the
    status, and nest checks.
    """

    ok: bool = True

    @contextmanager
    def check(self, name: str, clear_on_success: bool = True) -> Generator['CheckResult', None, None]:
        """check returns a context manager that handles printing a '[....]'  /
        '[ OK ]' / '[FAIL]' check.  While the check is running, it
        will stream whatever you write to stdout/stderr.  If
        clear_on_success is True, then once the check finishes, if the
        check passed then it will erase that stdout/stderr output,
        since you probably only want diagnostic output if the check
        fails.
        You can provide a (1-line) textual check result that will be
        shown on both success and failure by writing to "mycheck.result".
        You may cause a check to fail by either raising an Exception,
        or by setting "mycheck.ok = False".  If you do neither of these,
        then the check will be considered to pass.
        The mycheck.subcheck method returns a context manager for a
        nested child check.
        """
        def line(status: str, rest: Optional[str] = None) -> str:
            txt = name
            if rest:
                txt = f'{txt}: {rest}'
            return f" {status}{ansiterm.sgr} {txt}"

        output = LineTracker(output=sys.stdout)
        output.write(line(status=f'{ansiterm.sgr.bold.fg_blu}[....]') + "\n")

        check = CheckResult()

        with capture_output(Indent(output=output, indent="   > ")):
            try:
                yield check
            except Exception as err:
                if str(err).strip():
                    print(err)
                check.ok = False

        end = output.cur_line
        output.goto_line(1)
        if check.ok:
            output.write(line(status=f'{ansiterm.sgr.bold.fg_grn}[ OK ]', rest=check.result))
        else:
            output.write(line(status=f'{ansiterm.sgr.bold.fg_red}[FAIL]', rest=check.result))
        if check.ok and clear_on_success:
            output.write(ansiterm.clear_rest_of_screen + "\n")
        else:
            output.write(ansiterm.clear_rest_of_line)
            output.goto_line(end)

        self.ok &= check.ok

    # alias for readability
    subcheck = check


class CheckResult(Checker):
    """A CheckResult is the context manager type returned by
    "Checker.check".
    """
    result: Optional[str] = None
