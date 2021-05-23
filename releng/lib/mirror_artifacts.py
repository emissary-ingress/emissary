#!/hint/python3

import shlex
import subprocess
from contextlib import contextmanager
from typing import Generator, List

from .uiutil import run as _run
from .uiutil import run_txtcapture


def run(args: List[str]) -> None:
    print("$ " + (" ".join(shlex.quote(arg) for arg in args)))
    _run(args)


@contextmanager
def gcr_login() -> Generator[None, None, None]:
    key = run_txtcapture(
        ['keybase', 'fs', 'read', '/keybase/team/datawireio/secrets/googlecloud.gcr-ci-robot.datawire.json.key'])
    subprocess.run(['docker', 'login', '-u', '_json_key', '--password-stdin', 'https://gcr.io'],
                   check=True,
                   text=True,
                   input=key)
    yield
    subprocess.run(['docker', 'logout', 'https://gcr.io'], check=True)


def main(tags: List[str],
         source_registry: str = 'docker.io/datawire',
         repos: List[str] = ['ambassador',],
         image_append: str = '') -> None:
    with gcr_login():
        for repo in repos:
            for tag in tags:
                run(['docker', 'pull', f'{source_registry}/{repo}:{tag}'])
                for registry in ['quay.io/datawire', 'gcr.io/datawire']:
                    src = f'{source_registry}/{repo}:{tag}'
                    dst = f'{registry}/{repo}:{tag}'
                    if dst == src:
                        continue
                    if image_append != '':
                        dst = f'{registry}/{repo}-{image_append}:{tag}'
                    run(['docker', 'tag', src, dst])
                    run(['docker', 'push', dst])
