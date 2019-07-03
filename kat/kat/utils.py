import re
import subprocess

KAT_FAMILY='skinningkats'

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
    def __init__(self, what, *args, defer: bool=False, fake: bool=False, verbose: bool=False, may_fail: bool=False, **kwargs) -> None:
        self.what = what
        self.defer = defer
        self.fake = fake
        self.verbose = verbose
        self.may_fail = may_fail

        for arg in "stdout", "stderr":
            if arg not in kwargs:
                kwargs[arg] = subprocess.PIPE

        self.args = args
        self.kwargs = kwargs

        self.cmdline = " ".join([quote(x) for x in self.args])

        if self.defer:
            if self.verbose:
                print(f'---- deferring: {self.cmdline}')
        else:
            self.start()

    def start(self) -> None:
        if self.verbose:
            verb = 'faking' if self.fake else 'running'

            print(f'---- {verb}: {self.cmdline}')

        if not self.fake:
            self.proc = subprocess.run(self.args, **self.kwargs)

    def check(self) -> bool:
        if self.fake:
            return True

        try:
            self.proc.check_returncode()
            return True
        except Exception as e:
            if self.verbose or not self.may_fail:
                print(f"==== COMMAND FAILED: {self.what}")
                print("---- command line ----")
                print(self.cmdline)
                print("---- stdout ----")
                print(self.stdout)
                print("")
                print("---- stderr ----")
                print(self.stderr)

            if self.may_fail:
                return True
            else:
                return False

    @property
    def stdout(self) -> str:
        return self.proc.stdout.decode("utf-8")

    @property
    def stderr(self) -> str:
        return self.proc.stderr.decode("utf-8")

    @classmethod
    def run(cls, what: str, *args, **kwargs) -> bool:
        return ShellCommand(what, *args, **kwargs).check()
