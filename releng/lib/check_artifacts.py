#!/hint/python3

import json
import os
import os.path
import sys
from contextlib import contextmanager
from typing import Dict, Generator, Optional, Tuple, cast
from urllib.error import HTTPError
from urllib.request import urlopen

from . import ansiterm, assert_eq, build_version, get_is_private
from .uiutil import Checker, CheckResult, run, run_bincapture, run_txtcapture


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


def main(rc_ver: str, ga: bool, include_latest: bool, include_docker: bool = True) -> int:
    warning = """
 ==> Warning: FIXME: While this script is handy in the things that it
     does check, there's still quite a bit that it doesn't check;
     check_artifacts.py is still riddled with "TODO"s.  Don't be
     lulled in to thinking that running this script means you don't
     need to do anything else.
"""
    print(f"{ansiterm.sgr.fg_red}{warning}{ansiterm.sgr}")

    ga_ver = build_version(rc_ver)

    is_private = get_is_private()

    def do_check_docker(checker: Checker, name: str) -> None:
        with checker.check(name=f'Docker image: {name}', clear_on_success=False) as check:
            iids = []
            if is_private:
                registries = ['quay.io/datawire-private']
            else:
                registries = ['docker.io/datawire', 'quay.io/datawire', 'gcr.io/datawire']
                registries = ['docker.io/alixcook11']
            for registry in registries:
                tags = [rc_ver]
                if ga:
                    tags = [ga_ver] + tags
                for tag in tags:
                    fulltag = f'{registry}/{name}:{tag}'
                    with check.subcheck(name=fulltag) as subcheck:
                        iid = docker_pull(fulltag)
                        iids += [iid]
                        subcheck.result = iid[len('sha256:'):len('sha256:') + 12]
            with check.subcheck(name='All images match') as subcheck:
                if len(iids) == 0:
                    return
                a = iids[0]
                for b in iids[1:]:
                    if b != a:
                        subcheck.ok = False

    def do_check_binary(checker: Checker, name: str, txt: bool, private: bool) -> None:
        with checker.check(name=f'Executable: {name}', clear_on_success=False) as checker:
            for platform in ['linux/amd64/{}', 'darwin/amd64/{}', 'windows/amd64/{}.exe']:
                rc_body: Optional[bytes] = None
                with do_check_s3(checker, f'{name}/{rc_ver}/{platform.format(name)}',
                                 private=private) as (subcheck, body):
                    if body is not None:
                        rc_body = body
                        # TODO: Validate the binary somehow
                if ga:
                    with do_check_s3(checker, f'{name}/{ga_ver}/{platform.format(name)}',
                                     private=private) as (subcheck, body):
                        if body is not None:
                            assert body == rc_body
            if txt:
                if include_latest:
                    with do_check_s3(checker, f'{name}/latest.txt', private=private) as (subcheck, body):
                        if body is not None:
                            subcheck.result = body.decode('UTF-8').strip()
                            if is_private:
                                assert subcheck.result != rc_ver
                            else:
                                assert_eq(subcheck.result, rc_ver)
                if ga or is_private:
                    with do_check_s3(checker, f'{name}/stable.txt', private=private) as (subcheck, body):
                        if body is not None:
                            subcheck.result = body.decode('UTF-8').strip()
                            if is_private:
                                assert subcheck.result != ga_ver
                            else:
                                assert_eq(subcheck.result, ga_ver)

    s3_login()

    checker = Checker()

    if include_docker:
        do_check_docker(checker, 'ambassador')
        with checker.check('Ambassador S3 files', clear_on_success=False) as checker:
            with do_check_s3(checker, name='ambassador/teststable.txt') as (subcheck, body):
                if body is not None:
                    subcheck.result = body.decode('UTF-8').strip()
                    if is_private:
                        assert subcheck.result != rc_ver
                    else:
                        assert_eq(subcheck.result, rc_ver)
            if ga or is_private:
                with do_check_s3(checker, name='ambassador/stable.txt') as (subcheck, body):
                    if body is not None:
                        subcheck.result = body.decode('UTF-8').strip()
                        if is_private:
                            assert subcheck.result != ga_ver
                        else:
                            assert_eq(subcheck.result, ga_ver)
            with do_check_s3(checker, name='ambassador/testapp.json', bucket='scout-datawire-io',
                             private=True) as (subcheck, body):
                if body is not None:
                    subcheck.result = json.loads(body.decode('UTF-8')).get('latest_version', '')
                    if is_private:
                        assert subcheck.result != rc_ver
                    else:
                        assert_eq(subcheck.result, rc_ver)
            if ga or is_private:
                with do_check_s3(checker, name='ambassador/app.json', bucket='scout-datawire-io',
                                 private=True) as (subcheck, body):
                    if body is not None:
                        subcheck.result = json.loads(body.decode('UTF-8')).get('latest_version', '')
                        if is_private:
                            assert subcheck.result != ga_ver
                        else:
                            assert_eq(subcheck.result, ga_ver)

    if not ga:
        with checker.check(name='Git tags') as check:
            check.result = 'TODO'
            raise NotImplementedError()
        with checker.check(name='Pull Requests') as check:
            check.result = 'TODO'
            raise NotImplementedError()
    else:
        with checker.check(name='Git tags') as check:
            check.result = 'TODO'
            raise NotImplementedError()
        with checker.check(name='Website YAML') as check:
            yaml_str = http_cat('https://getambassador.io/yaml/ambassador/ambassador-rbac.yaml').decode('utf-8')
            images = [
                line.strip()[len('image:'):].strip() for line in yaml_str.split("\n")
                if line.strip().startswith('image:')
            ]
            assert_eq(len(images), 2)   # One for Ambassador, one for the Agent.

            for image in images:
                assert '/ambassador:' in image
                check.result = image.split(':', 1)[1]
                assert_eq(check.result, ga_ver)
        with checker.check(name='Helm Chart') as check:
            run(['helm', 'repo', 'add', 'datawire', 'https://getambassador.io'])
            run(['helm', 'repo', 'update'])
            yaml_str = run_txtcapture(['helm', 'show', 'chart', 'datawire/ambassador'])
            versions = [
                line[len('appVersion:'):].strip() for line in yaml_str.split("\n") if line.startswith('appVersion:')
            ]
            assert_eq(len(versions), 1)
            check.result = versions[0]
            assert_eq(check.result, ga_ver)
        with checker.check(name='ambassador.git GitHub release for chart') as check:
            check.result = 'TODO'
            raise NotImplementedError()
        with checker.check(name='ambassador.git GitHub release for code') as check:
            check.result = 'TODO'
            raise NotImplementedError()

    if not checker.ok:
        return 1
    return 0
