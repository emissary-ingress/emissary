from .uiutil import run

def git_add(filename: str) -> None:
    run(['git', 'add', '--', filename])
