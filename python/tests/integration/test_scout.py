from typing import Any, Optional

import os
import time
import sys

import pexpect
import pytest
import requests

from tests.runutils import run_and_assert

DOCKER_IMAGE = os.environ.get("AMBASSADOR_DOCKER_IMAGE", None)

child: Optional[pexpect.spawnbase.SpawnBase] = None           # see docker_start()
diagd_host: Optional[str] = None # see docker_start()
child_name = "diagd-unset"      # see docker_start() and docker_kill()

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

    global child_name
    child_name = f"diagd-{int(time.time() * 1000)}"

    global diagd_host

    cmd = f'docker run --name {child_name} --rm --entrypoint=dev-magic-entrypoint -p 9999:9999 {DOCKER_IMAGE}'
    diagd_host = 'localhost:9999'

    child = pexpect.spawn(cmd, encoding='utf-8')
    child.logfile = logfile

    i = child.expect([ pexpect.EOF, pexpect.TIMEOUT, 'LocalScout: mode boot, action boot1' ])

    if i == 0:
        print('diagd died?')
        return False
    elif i == 1:
        print('diagd timed out?')
        return False

    # Set up port forwarding in the Ambassador container from all:9999, where
    # this test will connect, to localhost:9998, where diagd is listening. This
    # is necessary because diagd rejects (403) requests originating outside the
    # container for security reasons.

    # Copy the simple port forwarding script into the container
    child2 = pexpect.spawn(f'docker cp python/tests/_forward.py {child_name}:/tmp/', encoding='utf-8')
    child2.logfile = logfile

    if child2.expect([ pexpect.EOF, pexpect.TIMEOUT ]) == 1:
        print("docker cp timed out?")
        return False

    child2.close()
    if child2.exitstatus != 0:
        print("docker cp failed?")
        return False

    # Run the port forwarding script
    child2 = pexpect.spawn(f'docker exec -d {child_name} python /tmp/_forward.py localhost 9998 "" 9999', encoding='utf-8')
    child2.logfile = logfile

    if child2.expect([ pexpect.EOF, pexpect.TIMEOUT ]) == 1:
        print("docker exec timed out?")
        return False

    child2.close()
    if child2.exitstatus != 0:
        print("docker exec failed?")
        return False

    return True

def docker_kill(logfile):
    cmd = f'docker kill {child_name}'

    child = pexpect.spawn(cmd, encoding='utf-8')
    child.logfile = logfile

    child.expect([ pexpect.EOF, pexpect.TIMEOUT ])

def wait_for_diagd(logfile) -> bool:
    status = False
    tries_left = 5

    while tries_left >= 0:
        logfile.write(f'...checking diagd ({tries_left})\n')

        try:
            global diagd_host
            response = requests.get(f'http://{diagd_host}/_internal/v0/ping',
                                    headers={ "X-Ambassador-Diag-IP": "127.0.0.1" })

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
        global diagd_host
        response = requests.post(f'http://{diagd_host}/_internal/v0/fs',
                                 headers={ "X-Ambassador-Diag-IP": "127.0.0.1" },
                                 params={ 'path': f'cmd:{cmd}' })
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
        global diagd_host
        response = requests.get(f'http://{diagd_host}/_internal/v0/events',
                                headers={ "X-Ambassador-Diag-IP": "127.0.0.1" })

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

@pytest.mark.flaky(reruns=1, reruns_delay=10)
def test_scout():
    test_status = False

    with open('/tmp/test_scout_output', 'w') as logfile:
        if not DOCKER_IMAGE:
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
