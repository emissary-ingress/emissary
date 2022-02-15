#!/hint/python3

import json
import os
import os.path
import sys
from contextlib import contextmanager
from typing import Dict, Generator, Optional, Tuple, cast
from urllib.error import HTTPError
from urllib.request import urlopen
import fileinput
import subprocess

from . import ansiterm, assert_eq, build_version, get_is_private
from .uiutil import Checker, CheckResult, run, run_bincapture, run_txtcapture
from .mirror_artifacts import get_images


def docker_pull(tag: str) -> str:
    """`docker pull` and then return the image ID"""
    run(['docker', 'pull', tag])
    return run_txtcapture(['docker', 'image', 'inspect', tag, '--format={{.Id}}'])


def s3_login() -> None:
    cred_str = run_txtcapture(
        ['keybase', 'fs', 'read', '/keybase/team/datawireio/secrets/aws.datawire-release-bot.access-key-id'])
    for line in cred_str.split("\n"):
        k, v = line.split(':')
        os.environ[f"AWS_{k.strip()}"] = v.strip()


def s3_cat(url: str) -> bytes:
    return run_bincapture(['aws', 's3', 'cp', url, '-'])


def http_cat(url: str) -> bytes:
    with urlopen(url) as fh:
        return cast(bytes, fh.read())  # docs say .read() returns 'bytes', typeshed says it returns 'Any'?


@contextmanager
def do_check_s3(checker: Checker,
                name: str,
                bucket: str = 'datawire-static-files',
                private: bool = False) -> Generator[Tuple[CheckResult, Optional[bytes]], None, None]:
    prefix: Dict[bool, str] = {
        True: f's3://{bucket}/',
        False: f'https://s3.amazonaws.com/{bucket}/',
    }
    url = prefix[private] + name
    with checker.check(name=url) as out:
        try:
            if private:
                publicly_readable = True
                try:
                    http_cat(prefix[False] + name)
                except HTTPError:
                    publicly_readable = False
                if publicly_readable:
                    raise Exception('Should be private, but is publicly readable')
                body = s3_cat(url)
            else:
                body = http_cat(url)
        except Exception as err:
            yield (out, None)
            raise
        else:
            yield (out, body)


def main(ga_ver: str, chart_ver: str, include_docker: bool = True,
        release_channel: str = "", source_registry: str ="docker.io/datawire", 
        image_append: str = "", image_name: str = "emissary",
        s3_bucket: str = "datawire-static-files") -> int:
    warning = """
 ==> Warning: FIXME: While this script is handy in the things that it
     does check, there's still quite a bit more that it could check. Don't
     be fooled into thinking that this script is complete.
"""
    print(f"{ansiterm.sgr.fg_red}{warning}{ansiterm.sgr}")


    is_private = get_is_private()

    def do_check_docker(checker: Checker, name: str) -> None:
        with checker.check(name=f'Docker image: {name}', clear_on_success=False) as check:
            iids = []
            if release_channel != '':
                tags = [f"{ga_ver}-{release_channel}"]
            else:
                tags = [ga_ver]

            for tag in tags:
                if is_private:
                    images = [f'quay.io/datawire-private/ambassador:{tag}']
                else:
                    images = get_images(source_registry, image_name, tag, image_append)
                for image in images:
                    with check.subcheck(name=image) as subcheck:
                        iid = docker_pull(image)
                        iids += [iid]
                        subcheck.result = iid[len('sha256:'):len('sha256:') + 12]
            with check.subcheck(name='All images match') as subcheck:
                if len(iids) == 0:
                    return
                a = iids[0]
                for b in iids[1:]:
                    if b != a:
                        subcheck.ok = False

    s3_login()

    checker = Checker()

    if include_docker:
        do_check_docker(checker, 'ambassador')
        with checker.check('Ambassador S3 files', clear_on_success=False) as checker:
            with do_check_s3(checker, name=f'emissary-ingress/{release_channel}stable.txt', bucket=s3_bucket) as (subcheck, body):
                if body is not None:
                    subcheck.result = body.decode('UTF-8').strip()
                    if is_private:
                        assert subcheck.result != ga_ver
                    else:
                        assert_eq(subcheck.result, ga_ver)
            with do_check_s3(checker, name=f'emissary-ingress/{release_channel}app.json', bucket='scout-datawire-io',
                             private=True) as (subcheck, body):
                if body is not None:
                    subcheck.result = json.loads(body.decode('UTF-8')).get('latest_version', '')
                    if is_private:
                        assert subcheck.result != ga_ver
                    else:
                        assert_eq(subcheck.result, ga_ver)

    with checker.check(name='Website YAML') as check:
        yaml_str = http_cat('https://app.getambassador.io/yaml/emissary/latest/emissary-emissaryns.yaml').decode('utf-8')
        images = [
            line.strip()[len('image:'):].strip() for line in yaml_str.split("\n")
            if line.strip().startswith('image:')
        ]
        assert_eq(len(images), 2)   # One for Ambassador, one for the Agent.

        check_tag = ga_ver
        if release_channel != '':
            check_tag = f"{check_tag}-{release_channel}"
        for image in images:
            assert '/emissary:' in image
            check.result = image.split(':', 1)[1]
            assert_eq(check.result, check_tag)

    with checker.check(name="Updating helm repo"):
        subprocess.run(['helm', 'repo', 'rm', 'emissary'], stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
        subprocess.run(['helm', 'repo', 'add', 'emissary',
                'https://s3.amazonaws.com/{}/charts'.format(s3_bucket)], stdout=subprocess.PIPE, stderr=subprocess.STDOUT)

        run(['helm', 'repo', 'update'])

    with checker.check(name="Check Helm Chart"):
        yaml_str = run_txtcapture(['helm', 'show', 'chart', '--version', chart_ver, 'emissary/emissary-ingress'])

        versions = [
            line[len('appVersion:'):].strip() for line in yaml_str.split("\n") if line.startswith('appVersion:')
        ]
        assert_eq(len(versions), 1)
        check.result = versions[0]
        check_tag = ga_ver
        if release_channel != '':
            check_tag = f"{check_tag}-{release_channel}"
        assert_eq(check.result, check_tag)

    # The existence of a GitHub release implies the existence of its tag, and we check to
    # make sure that the tag matches what we expect. Therefore we don't do a separate check
    # for the tag. (It's true that you can delete the tag after the release; we're just not
    # going to worry about that.)
    with checker.check(name='ambassador.git GitHub release for chart (implies GitHub tag, too)') as check:
        tag = run_txtcapture([
            "gh", "release", "view",
            "--json=tagName",
            "--jq=.tagName",
            "--repo=emissary-ingress/emissary",
            f"chart/v{chart_ver}"])
        assert_eq(tag.strip(), f"chart/v{chart_ver}")

    # See above re tags.
    with checker.check(name='ambassador.git GitHub release for code (implies GitHub tag, too)') as check:
        tag = run_txtcapture([
            "gh", "release", "view",
            "--json=tagName",
            "--jq=.tagName",
            "--repo=emissary-ingress/emissary",
            f"v{ga_ver}"])
        assert_eq(tag.strip(), f"v{ga_ver}")

    if not checker.ok:
        return 1
    return 0
