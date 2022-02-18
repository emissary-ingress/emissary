#!/hint/python3

import os
import shlex
import subprocess
from contextlib import contextmanager
from typing import Generator, Iterable, List

from .uiutil import run as _run
from .uiutil import run_txtcapture


def run(args: List[str], /) -> None:
    print("$ " + (" ".join(shlex.quote(arg) for arg in args)))
    _run(args)


@contextmanager
def gcr_login() -> Generator[None, None, None]:
    key = run_txtcapture(
        ['keybase', 'fs', 'read', '/keybase/team/datawireio/secrets/googlecloud.gcr-ci-robot.datawire.json.key'])

    subprocess.run(
        ['gcloud', 'auth', 'activate-service-account', '--key-file=-'],
        check=True,
        text=True,
        input=key,)
    subprocess.run(['gcloud', 'auth', 'configure-docker'], check=True)
    yield
    subprocess.run(['docker', 'logout', 'https://gcr.io'], check=True)


default_repos = {
    'docker.io/emissaryingress/emissary',
    'gcr.io/datawire/emissary',
}

default_source_repo = 'docker.io/emissaryingress/emissary'


def enumerate_images(*, repos: Iterable[str] = default_repos, tag: str) -> Iterable[str]:
    return [f"{repo}:{tag}" for repo in repos]


def mirror_images(*, repos: Iterable[str] = default_repos, tag: str, source_repo: str = default_source_repo) -> None:
    print('Note: This script can be rerun.')
    print('If pushes to registries fail, you can rerun the command in your terminal to debug.')
    print('If pushes fail, it might be a credentials problem with gcr or quay.io or an issue with your gcloud installation.')

    with gcr_login():
        src = f'{source_repo}:{tag}'
        dsts = enumerate_images(repos=repos, tag=tag)

        run(['docker', 'pull', src])
        for dst in dsts:
            if dst == src:
                continue
            run(['docker', 'tag', src, dst])
            run(['docker', 'push', dst])
