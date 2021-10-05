#!python

# Copyright 2019-2020 Datawire. All rights reserved.
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
# limitations under the License

########
# This is a debugging tool that can grab snapshots and Envoy configs from
# Ambassador's configuration directory, sanitize secrets out of the snapshots,
# and hand back a compressed tarfile that the user can hand back to Datawire.
########

import sys

import functools
import glob
import json
import os
import tarfile

import click

from ambassador.utils import dump_json

# Use this instead of click.option
click_option = functools.partial(click.option, show_default=True)
click_option_no_default = functools.partial(click.option, show_default=False)


def sanitize_snapshot(snapshot: dict):
    sanitized = {}

    # Consul is pretty easy. Just sort, using service-dc as the sort key.
    consul_elements = snapshot.get('Consul')

    if consul_elements:
        csorted = {}

        for key, value in consul_elements.items():
            csorted[key] = sorted(value, key=lambda x: f'{x["Service"]-x["Id"]}')

        sanitized['Consul'] = csorted

    # Make sure we grab Deltas and Invalid -- these should really be OK as-is.

    for key in [ 'Deltas', 'Invalid' ]:
        if key in snapshot:
            sanitized[key] = snapshot[key]

    # Kube is harder because we need to sanitize Kube secrets.
    kube_elements = snapshot.get('Kubernetes')

    if kube_elements:
        ksorted = {}

        for key, value in kube_elements.items():
            if not value:
                continue

            if key == 'secret':
                for secret in value:
                    if "data" in secret:
                        data = secret["data"]

                        for k in data.keys():
                            data[k] = f'-sanitized-{k}-'

                    metadata = secret.get('metadata', {})
                    annotations = metadata.get('annotations', {})

                    # Wipe the last-applied-configuration annotation, too, because it
                    # often contains the secret data.
                    if 'kubectl.kubernetes.io/last-applied-configuration' in annotations:
                        annotations['kubectl.kubernetes.io/last-applied-configuration'] = '--sanitized--'

            # All the sanitization above happened in-place in value, so we can just
            # sort it.
            ksorted[key] = sorted(value, key=lambda x: x.get('metadata',{}).get('name'))

        sanitized['Kubernetes'] = ksorted

    return sanitized


# Helper to open a snapshot.yaml and sanitize it.
def helper_snapshot(path: str) -> str:
    snapshot = json.loads(open(path, "r"). read())

    return dump_json(sanitize_snapshot(snapshot))


# Helper to open a problems.json and sanitize the snapshot it contains.
def helper_problems(path: str) -> str:
    bad_dict = json.loads(open(path, "r"). read())

    bad_dict["snapshot"] = sanitize_snapshot(bad_dict["snapshot"])

    return dump_json(bad_dict)


# Helper to just copy a file.
def helper_copy(path: str) -> str:
    return open(path, "r").read()


# Open a tarfile for output...
@click.command(help="Grab, and sanitize, Ambassador snapshots for later debugging")
@click_option('--debug/--no-debug', default=True,
              help="enable debugging")
@click_option('-o', '--output-path', '--output', type=click.Path(writable=True), default="sanitized.tgz",
              help="output path")
@click_option('-s', '--snapshot-dir', '--snapshot', type=click.Path(exists=True, dir_okay=True, file_okay=False),
              help="snapshot directory to read")
def main(snapshot_dir: str, debug: bool, output_path: str) -> None:
    if not snapshot_dir:
        config_base_dir = os.environ.get("AMBASSADOR_CONFIG_BASE_DIR", "/ambassador")
        snapshot_dir = os.path.join(config_base_dir, "snapshots")

    if debug:
        print(f"Saving sanitized snapshots from {snapshot_dir} to {output_path}")

    with tarfile.open(output_path, 'w:gz') as archive:
        # ...then iterate any snapshots, sanitize, and stuff 'em in the tarfile.
        # Note that the '.yaml' on the snapshot file name is a misnomer: when
        # watt is involved, they're actually JSON. It's a long story.

        some_found = False

        interesting_things = [
            ( "snap*yaml", helper_snapshot ),
            ( "problems*json", helper_problems ),
            ( "econf*json", helper_copy ),
            ( "diff*txt", helper_copy )
        ]

        for pattern, helper in interesting_things:
            for path in glob.glob(os.path.join(snapshot_dir, pattern)):
                some_found = True

                # The tarfile can be flat, rather than embedding everything
                # in a directory with a fixed name.
                b = os.path.basename(path)

                if debug:
                    print(f"...{b}")

                sanitized = helper(path)

                if sanitized:
                    _, ext = os.path.splitext(path)
                    sanitized_name = f"sanitized{ext}"

                    with open(sanitized_name, 'w') as tmp:
                        tmp.write(sanitized)

                    archive.add(sanitized_name, arcname=b)
                    os.unlink(sanitized_name)

        if not some_found:
            sys.stderr.write(f"No snapshots found in {snapshot_dir}?\n")
            sys.exit(1)

        sys.exit(0)

if __name__ == "__main__":
    main()
