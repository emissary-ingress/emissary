from typing import Any, Dict, List, Optional

import os
import time
import sys

import pexpect
import pytest
import requests

DockerImage = os.environ.get("AMBASSADOR_DOCKER_IMAGE", None)
child = None    # see docker_start()

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

def docker_start(logfile) -> bool:
    # Use a global here so that the child process doesn't get killed
    global child

    cmd = f'docker run --rm --network {os.environ["DOCKER_NETWORK"]} --network-alias diagd {os.environ["AMBASSADOR_DOCKER_IMAGE"]} --dev-magic'

    child = pexpect.spawn(cmd, encoding='utf-8')
    child.logfile = logfile

    i = child.expect([ pexpect.EOF, pexpect.TIMEOUT, 'LocalScout: mode boot, action boot1' ])

    if i == 0:
        print('diagd died?')
        return False
    elif i == 1:
        print('diagd timed out?')
        return False
    else:
        return True

def docker_kill(logfile):
    cmd = f'docker kill diagd'

    child = pexpect.spawn(cmd, encoding='utf-8')
    child.logfile = logfile

    child.expect([ pexpect.EOF, pexpect.TIMEOUT ])

def wait_for_diagd(logfile) -> bool:
    status = False
    tries_left = 5

    while tries_left >= 0:
        logfile.write(f'...checking diagd ({tries_left})\n')

        try:
            response = requests.get('http://diagd:9998/_internal/v0/ping')

            if response.status_code == 200:
                logfile.write('   got it\n')
                status = True
                break
            else:
                logfile.write(f'   failed {response.status_code}\n')
        except requests.exceptions.RequestException as e:
            logfile.write(f'   failed {e}\n')

        tries_left -= 1
        time.sleep(2)

    return status

def check_http(logfile, cmd: str) -> bool:
    try:
        response = requests.post('http://diagd:9998/_internal/v0/fs', params={ 'path': f'cmd:{cmd}' })
        text = response.text

        if response.status_code != 200:
            logfile.write(f'{cmd}: wanted 200 but got {response.status_code} {text}\n')
            return False

        return True
    except Exception as e:
        logfile.write(f'Could not do HTTP: {e}\n')

        return False

def fetch_events(logfile) -> Any:
    try:
        response = requests.get('http://diagd:9998/_internal/v0/events')

        if response.status_code != 200:
            logfile.write(f'events: wanted 200 but got {response.status_code} {response.text}\n')
            return None

        data = response.json()

        return data
    except Exception as e:
        logfile.write(f'events: could not do HTTP: {e}\n')

        return None

def check_chimes(logfile) -> bool:
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
        logfile.write(f'RESETTING for sequence {i}\n')

        if not check_http(logfile, 'chime_reset'):
            logfile.write(f'could not reset for sequence {i}\n')
            result = False
            continue

        j = 0
        for cmd in cmds:
            logfile.write(f'   sending {cmd} for sequence {i}.{j}\n')

            if not check_http(logfile, cmd):
                logfile.write(f'could not do {cmd} for sequence {i}.{j}\n')
                result = False
                break

            j += 1

        if not result:
            continue

        events = fetch_events(logfile)

        if not events:
            result = False
            continue

        # logfile.write(json.dumps(events, sort_keys=True, indent=4))

        logfile.write('   ----\n')
        verdict = []

        for timestamp, mode, action, data in events:
            verdict.append(action)

            action_key = data.get('action_key', None)

            if action_key:
                covered[action_key] = True

            logfile.write(f'     {action} - {action_key}\n')

        # logfile.write(json.dumps(verdict, sort_keys=True, indent=4\n))

        if verdict != wanted_verdict:
            logfile.write(f'verdict mismatch for sequence {i}:\n')
            logfile.write(f'  wanted {" ".join(wanted_verdict)}\n')
            logfile.write(f'  got    {" ".join(verdict)}\n')

        i += 1

    for key in sorted(covered.keys()):
        if not covered[key]:
            logfile.write(f'missing coverage for {key}\n')
            result = False

    return result

def test_scout():
    test_status = False

    with open('/tmp/test_scout_output', 'w') as logfile:
        if not DockerImage:
            logfile.write('No $AMBASSADOR_DOCKER_IMAGE??\n')
        else:
            if docker_start(logfile):
                if wait_for_diagd(logfile) and check_chimes(logfile):
                    test_status = True

                docker_kill(logfile)

    if not test_status:
        with open('/tmp/test_scout_output', 'r') as logfile:
            for line in logfile:
                print(line.rstrip())

    assert test_status, 'test failed'

if __name__ == '__main__':
    pytest.main(sys.argv)
