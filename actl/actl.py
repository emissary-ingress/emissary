# Copyright 2017 Datawire. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""
Command line helper for Ambassador
"""

import shlex
import subprocess
import sys
import time

import click
import requests


CONTEXT_SETTINGS = dict(help_option_names=["-h", "--help"])
VERSION = "0.0"
VERBOSE = False


@click.group(context_settings=CONTEXT_SETTINGS)
@click.version_option(version=VERSION)
@click.option("--verbose", "-v", is_flag=True, default=False)
def actl(verbose):
    """Command line tool to interact with Ambassador"""
    global VERBOSE
    VERBOSE = verbose


def call_command(cmd, do_exit=True, **kwargs):
    if str(cmd) == cmd:
        cmd = shlex.split(cmd)
    if VERBOSE:
        print("==> " + " ".join(cmd), file=sys.stderr)
    res = subprocess.run(cmd, universal_newlines=True, **kwargs)
    if res.returncode == 0:
        return res
    print(f"==> Call failed with exit code {res.returncode}", file=sys.stderr)
    if do_exit:
        exit(res.returncode)
    return None


def read_from_command(cmd, **kwargs):
    process = call_command(cmd, stdout=subprocess.PIPE, **kwargs)
    return process.stdout


def get_ambassador_pod_name():
    """Get the name of the (First) ambassador pod"""
    cmd = "kubectl get pod -l service=ambassador -o jsonpath={.items[0].metadata.name}"
    return read_from_command(cmd)


def get_ambassador_url():
    kubecontext = read_from_command("kubectl config current-context")
    if kubecontext == "minikube":
        return read_from_command("minikube service --url ambassador")
    inspect = "kubectl get service ambassador --output jsonpath={%s}"
    while True:
        host = read_from_command(inspect % ".status.loadBalancer.ingress[0].hostname", do_exit=False)
        if host is None:
            host = read_from_command(inspect % ".status.loadBalancer.ingress[0].ip", do_exit=False)
        if host is not None:
            port = read_from_command(inspect % ".spec.ports[0].port", do_exit=False)
            if port is not None:
                if port == "443":
                    protocol = "https"
                else:
                    protocol = "http"
                return f"{protocol}://{host}"
        time.sleep(1.0)


@actl.command()
def geturl():
    """Emit export AMBASSADORURL shell snippet"""
    print(f"export AMBASSADORURL={get_ambassador_url()}")


@actl.command(name="map")
@click.argument("mapping")
@click.argument("prefix")
@click.argument("service")
@click.argument("rewrite", required=False, default="/")
def add_mapping(mapping, prefix, service, rewrite):
    """Map a resource to a service"""
    url = get_ambassador_url() + f"/ambassador/mapping/{mapping}"
    json = dict(prefix=prefix, service=service, rewrite=rewrite)
    if VERBOSE:
        print(f"==> POST {url}")
    response = requests.post(url, json=json)
    if response.status_code != 200:
        exit(f"Mapping attempt failed with status {response.status_code} {response.reason}")
    print(response.text)


@actl.command(name="unmap")
@click.argument("mapping")
def remove_mapping(mapping):
    """Remove an existing mapping"""
    url = get_ambassador_url() + f"/ambassador/mapping/{mapping}"
    if VERBOSE:
        print(f"==> DELETE {url}")
    response = requests.delete(url)
    if response.status_code != 200:
        exit(f"Unmapping attempt failed with status {response.status_code} {response.reason}")
    print(response.text)


@actl.command(name="mappings")
def list_mappings():
    """List current mappings"""
    url = get_ambassador_url() + "/ambassador/mappings"
    if VERBOSE:
        print(f"==> GET {url}")
    response = requests.get(url)
    if response.status_code != 200:
        exit(f"List mappings attempt failed with status {response.status_code} {response.reason}")
    print(response.text)


def hand_off_to_command(cmd, **kwargs):
    try:
        call_command(cmd, **kwargs)  # Maybe this should be an exec(...) instead
    except KeyboardInterrupt:
        pass


@actl.command()
def forward():
    """Forward 8888 to admin interface"""
    pod_name = get_ambassador_pod_name()
    cmd = f"kubectl port-forward {pod_name} 8888"
    hand_off_to_command(cmd)


@actl.command()
def logs():
    """Show Ambassador's log output"""
    pod_name = get_ambassador_pod_name()
    cmd = f"kubectl logs {pod_name} -f -c ambassador"
    hand_off_to_command(cmd)


@actl.command()
@click.argument("command", default="/bin/bash", required=False)
def shell(command="/bin/bash"):
    """Run COMMAND on Ambassador's container [default: /bin/bash]"""
    pod_name = get_ambassador_pod_name()
    cmd = f"kubectl exec -it {pod_name} -c ambassador {command}"
    hand_off_to_command(cmd)


if __name__ == "__main__":
    actl()
