# Copyright 2018 Datawire. All rights reserved.
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

import click
import logging
import os
import uuid

from kubernetes import client, config, watch
from kubernetes.client.rest import ApiException

from ambassador.VERSION import Version

__version__ = Version
ambassador_id = os.getenv("AMBASSADOR_ID", "default")
ambassador_namespace = os.environ.get('AMBASSADOR_NAMESPACE', 'default')
ambassador_single_namespace = bool("AMBASSADOR_SINGLE_NAMESPACE" in os.environ)

logging.basicConfig(
    level=logging.INFO,  # if appDebug else logging.INFO,
    format="%%(asctime)s kubewatch [%%(process)d T%%(threadName)s] %s %%(levelname)s: %%(message)s" % __version__,
    datefmt="%Y-%m-%d %H:%M:%S"
)

# logging.getLogger("datawire.scout").setLevel(logging.DEBUG)
logger = logging.getLogger("kubewatch")
logger.setLevel(logging.INFO)


def kube_v1():
    # Assume we got nothin'.
    k8s_api = None

    # XXX: is there a better way to check if we are inside a cluster or not?
    if "KUBERNETES_SERVICE_HOST" in os.environ:
        # If this goes horribly wrong and raises an exception (it shouldn't),
        # we'll crash, and Kubernetes will kill the pod. That's probably not an
        # unreasonable response.
        config.load_incluster_config()
        if "AMBASSADOR_VERIFY_SSL_FALSE" in os.environ:
            configuration = client.Configuration()
            configuration.verify_ssl=False
            client.Configuration.set_default(configuration)
        k8s_api = client.CoreV1Api()
    else:
        # Here, we might be running in docker, in which case we'll likely not
        # have any Kube secrets, and that's OK.
        try:
            config.load_kube_config()
            k8s_api = client.CoreV1Api()
        except FileNotFoundError:
            # Meh, just ride through.
            logger.info("No K8s")
            pass

    return k8s_api


@click.command()
@click.option("--debug", is_flag=True, help="Enable debugging")
def main(debug):
    if debug:
        logger.setLevel(logging.DEBUG)

    found = None
    root_id = None

    cluster_id = os.environ.get('AMBASSADOR_CLUSTER_ID', os.environ.get('AMBASSADOR_SCOUT_ID', None))
    wanted = ambassador_namespace if ambassador_single_namespace else "default"

    if cluster_id:
        found = "environment"
    else:
        logger.debug("looking up ID for namespace %s" % wanted)

        v1 = kube_v1()

        if v1:
            try:
                ret = v1.read_namespace(wanted)
                root_id = ret.metadata.uid
                found = "namespace %s" % wanted
            except ApiException as e:
                # This means our namespace wasn't found?
                logger.error("couldn't read namespace %s? %s" % (wanted, e))

            if not root_id:
                # OK, so we had a crack at this and something went wrong. Give up and hardcode
                # something.
                root_id = "00000000-0000-0000-0000-000000000000"
                found = "hardcoded ID"
        else:
            logger.error("could not connect to Kubernetes")

        # One way or the other, we need to generate an ID here.
        cluster_url = "d6e_id://%s/%s" % (root_id, ambassador_id)
        logger.debug("cluster ID URL is %s" % cluster_url)

        cluster_id = str(uuid.uuid5(uuid.NAMESPACE_URL, cluster_url)).lower()

    logger.debug("cluster ID is %s (from %s)" % (cluster_id, found))

    print(cluster_id)


if __name__ == "__main__":
    main()
