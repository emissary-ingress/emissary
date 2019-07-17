from typing import Any, Dict, List, Optional

import sys
import os
import time

import pexpect
import requests

DockerImage = os.environ["AMBASSADOR_DOCKER_IMAGE"]

SEQUENCES = [
    (
        [ 'env_ok', 'chime' ],
        [ 'boot1', 'now-healthy' ]
    ),
    (
        [ 'env_ok', 'chime', 'scout_cache_reset', 'chime' ],
        [ 'boot1', 'now-healthy', 'healthy' ]
    ),
    (
        [ 'env_ok', 'chime', 'env_bad', 'chime' ],
        [ 'boot1', 'now-healthy', 'now-unhealthy' ]
    ),
    (
        [ 'env_bad', 'chime' ],
        [ 'boot1', 'unhealthy' ]
    ),
    (
        [ 'env_bad', 'chime', 'chime', 'scout_cache_reset', 'chime' ],
        [ 'boot1', 'unhealthy', 'unhealthy' ]
    ),
    (
        [ 'chime', 'chime', 'chime', 'env_ok', 'chime', 'chime' ],
        [ 'boot1', 'unhealthy', 'now-healthy' ]
    ),
]

def docker_start() -> bool:
    cmd = f'docker run --rm --name diagd -p9998:9998 {os.environ["AMBASSADOR_DOCKER_IMAGE"]} --dev-magic'

    child = pexpect.spawn(cmd, encoding='utf-8')
    child.logfile = sys.stdout

    i = child.expect([ pexpect.EOF, pexpect.TIMEOUT, 'LocalScout: mode boot, action boot1' ])

    if i == 0:
        print('diagd died?')
        return False
    elif i == 1:
        print('diagd timed out?')
        return False
    else:
        return True

def docker_kill():
    cmd = f'docker kill diagd'

    child = pexpect.spawn(cmd, encoding='utf-8')
    child.logfile = sys.stdout

    child.expect([ pexpect.EOF, pexpect.TIMEOUT ])

def wait_for_diagd() -> bool:
    status = False
    tries_left = 5

    while tries_left >= 0:
        print(f'...checking diagd ({tries_left})')

        try:
            response = requests.get('http://localhost:9998/_internal/v0/ping')

            if response.status_code == 200:
                status = True
                break
            else:
                print(f'   failed {response.status_code}')
        except requests.exceptions.RequestException as e:
            print(f'   failed {e}')

        tries_left -= 1
        time.sleep(2)

    return status

def check_http(cmd: str) -> bool:
    try:
        response = requests.post('http://localhost:9998/_internal/v0/fs', params={ 'path': f'cmd:{cmd}' })
        text = response.text

        if response.status_code != 200:
            print(f'{cmd}: wanted 200 but got {response.status_code} {text}')
            return False

        return True
    except Exception as e:
        print(f'Could not do HTTP: {e}')

        return False

def fetch_events() -> Any:
    try:
        response = requests.get('http://localhost:9998/_internal/v0/events')

        if response.status_code != 200:
            print(f'events: wanted 200 but got {response.status_code} {response.text}')
            return False

        data = response.json()

        return data
    except Exception as e:
        print(f'events: could not do HTTP: {e}')

        return None

def check_chimes() -> bool:
    result = True

    i = 0

    covered = {
        'F-F-F': False,
        'F-F-T': False,
        # 'F-T-F': False,   # This particular key can never be generated
        # 'F-T-T': False,   # This particular key can never be generated
        'T-F-F': False,
        'T-F-T': False,
        'T-T-F': False,
        'T-T-T': False,
    }


    for cmds, wanted_verdict in SEQUENCES:
        print(f'RESETTING for sequence {i}')

        if not check_http('chime_reset'):
            print(f'could not reset for sequence {i}')
            result = False
            continue

        j = 0
        for cmd in cmds:
            print(f'   sending {cmd} for sequence {i}.{j}')

            if not check_http(cmd):
                print(f'could not do {cmd} for sequence {i}.{j}')
                result = False
                break

            j += 1

        if not result:
            continue

        events = fetch_events()

        if not events:
            result = False
            continue

        # print(json.dumps(events, sort_keys=True, indent=4))

        print('   ----')
        verdict = []

        for timestamp, mode, action, data in events:
            verdict.append(action)

            action_key = data.get('action_key', None)

            if action_key:
                covered[action_key] = True

            print(f'     {action} - {action_key}')

        # print(json.dumps(verdict, sort_keys=True, indent=4))

        if verdict != wanted_verdict:
            print(f'verdict mismatch for sequence {i}:')
            print(f'  wanted {" ".join(wanted_verdict)}')
            print(f'  got    {" ".join(verdict)}')

        i += 1

    for key in sorted(covered.keys()):
        if not covered[key]:
            print(f'missing coverage for {key}')
            result = False

    return result

def test_scout():
    test_status = True

    if not DockerImage:
        assert False, f'You must set $AMBASSADOR_DOCKER_IMAGE'
    else:
        assert docker_start(), 'diagd could not start'

        if not wait_for_diagd():
            test_status = False
        elif not check_chimes():
            test_status = False

        docker_kill()

        assert test_status, 'test failed'

if __name__ == '__main__':
    test_scout()

