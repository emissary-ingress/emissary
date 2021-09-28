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

    subprocess.run(
        ['gcloud', 'auth', 'activate-service-account', '--key-file=-'],
        check=True,
        text=True,
        input=key,)
    subprocess.run(['gcloud', 'auth', 'configure-docker'], check=True)
    yield
    subprocess.run(['docker', 'logout', 'https://gcr.io'], check=True)


def get_images(source_registry: str, repo: str, tag: str, image_append: str = ''):
    images = [f"{source_registry}/{repo}:{tag}",]
    for registry in ['quay.io/datawire', 'gcr.io/datawire']:
        dst = f'{registry}/{repo}:{tag}'
        if image_append != '':
            dst = f'{registry}/{repo}-{image_append}:{tag}'
        images.append(dst)
    return images


def main(tags: List[str],
         source_registry: str = 'docker.io/datawire',
         repos: List[str] = ['emissary',],
         image_append: str = '') -> None:
    print('Note: This script can be rerun.')
    print('If pushes to registries fail, you can rerun the command in your terminal to debug.')
    print('If pushes fail, it might be a credentials problem with gcr or quay.io or an issue with your gcloud installation.')
    with gcr_login():
        for repo in repos:
            for tag in tags:
                images = get_images(source_registry, repo, tag, image_append)
                src = f'{source_registry}/{repo}:{tag}'
                run(['docker', 'pull', src])
                for dst in images:
                    if dst == src:
                        continue
                    run(['docker', 'tag', src, dst])
                    run(['docker', 'push', dst])
