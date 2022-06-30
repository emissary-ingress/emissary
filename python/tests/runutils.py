import subprocess
import time


def run_and_assert(command, communicate=True):
    print(f"Running command {command}")
    output = subprocess.Popen(command, stdout=subprocess.PIPE)
    if communicate:
        stdout, stderr = output.communicate()
        print("STDOUT", stdout.decode("utf-8") if stdout is not None else None)
        print("STDERR", stderr.decode("utf-8") if stderr is not None else None)
        assert output.returncode == 0, "non-zero exit status: %d" % output.returncode
        return stdout.decode("utf-8") if stdout is not None else None
    return None


def run_with_retry(command, retries=0):
    print(f"Running command {command}")
    returncode = -1
    decoded = ""
    tries = 0
    max_tries = retries + 1
    while returncode != 0 and tries < max_tries:
        output = subprocess.Popen(command, stdout=subprocess.PIPE)
        if tries > 0:
            print("SLEEPING 5 seconds, TRIES=%d" % tries)
            time.sleep(5)
        stdout, stderr = output.communicate()
        print("STDOUT", stdout.decode("utf-8") if stdout is not None else None)
        print("STDERR", stderr.decode("utf-8") if stderr is not None else None)
        returncode = output.returncode
        decoded = stdout.decode("utf-8") if stdout is not None else None
        tries = tries + 1
    assert returncode == 0, "non-zero exit status: %d" % output.returncode
    return decoded
