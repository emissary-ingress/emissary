import re
import subprocess


_quote_pos = re.compile('(?=[^-0-9a-zA-Z_./\n])')

def quote(arg):
    r"""
    >>> quote('\t')
    '\\\t'
    >>> quote('foo bar')
    'foo\\ bar'
    """

    # This is the logic emacs uses
    if arg:
        return _quote_pos.sub('\\\\', arg).replace('\n', "'\n'")
    else:
        return "''"


class ShellCommand:
    def __init__(self, *args, **kwargs) -> None:
        self.verbose = kwargs.pop('verbose', False)

        for arg in "stdout", "stderr":
            if arg not in kwargs:
                kwargs[arg] = subprocess.PIPE

        self.cmdline = " ".join([quote(x) for x in args])

        if self.verbose:
            print(f'---- running: {self.cmdline}')

        self.proc = subprocess.run(args, **kwargs)

    def status(self) -> bool:
        try:
            self.proc.check_returncode()
            return True
        except Exception as e:
            return False

    def check(self, what: str) -> bool:
        if self.status():
            return True
        else:
            print(f"==== COMMAND FAILED: {what}")
            print("---- command line ----")
            print(self.cmdline)
            print("---- stdout ----")
            print(self.stdout)
            print("")
            print("---- stderr ----")
            print(self.stderr)

            return False

    @property
    def stdout(self) -> str:
        return self.proc.stdout.decode("utf-8")

    @property
    def stderr(self) -> str:
        return self.proc.stderr.decode("utf-8")

    @classmethod
    def run(cls, what: str, *args, **kwargs) -> bool:
        return ShellCommand(*args, **kwargs).check(what)
