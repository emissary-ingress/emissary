import http.client
import json

from .uiutil import run, check_command
from .uiutil import run_txtcapture as run_capture

def branch_exists(branch_name: str) -> bool:
    return check_command(["git", "rev-parse", "--verify", branch_name])


def has_open_pr(gh_repo: str, base: str, branchname: str) -> bool:
    conn = http.client.HTTPSConnection("api.github.com")
    conn.request("GET", f"/repos/{gh_repo}/pulls?base={base}", headers={"User-Agent":"python"})
    r1 = conn.getresponse()
    body = r1.read()
    json_body = json.loads(body)
    for pr_info in json_body:
        if pr_info.get('head',{}).get('ref') == branchname:
            # check that it is open
            if pr_info.get('state') == 'open':
                return True
    return False


def git_add(filename: str) -> None:
    """
    Use `git add` to stage a single file.
    """

    run(['git', 'add', '--', filename])


def git_check_clean(allow_staged: bool = False) -> None:
    """
    Use `git status --porcelain` to check if the working tree is dirty.
    If allow_staged is True, allow staged files, but no unstaged changes.
    """

    out = run_capture(['git', 'status', '--porcelain'])
    if out:
        # Can we allow staged changes?
        if not allow_staged:
            # Nope. ANY changes are unacceptable, so we can short-circuit
            # here.
            raise Exception(out)

        # If here, staged changes are OK, and unstaged changes are not.
        # In the porcelain output, staged changes start with a change
        # character followed by a space, and unstaged changes start with a
        # space followed by a change character. So any lines with a non-space
        # in the second column are a problem here.

        lines = out.split('\n')
        problems = [line for line in lines if line[1] != ' ']

        if problems:
            raise Exception("\n".join(problems))
