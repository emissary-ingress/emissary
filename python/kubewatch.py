#!/usr/bin/env python3

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

from pathlib import Path

from kubernetes import client, config
from kubernetes.client.rest import ApiException

from ambassador.VERSION import Version

__version__ = Version
ambassador_id = os.getenv("AMBASSADOR_ID", "default")
ambassador_namespace = os.environ.get('AMBASSADOR_NAMESPACE', 'default')
ambassador_single_namespace = bool("AMBASSADOR_SINGLE_NAMESPACE" in os.environ)
ambassador_basedir = os.environ.get('AMBASSADOR_CONFIG_BASE_DIR', '/ambassador')

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
            configuration.verify_ssl = False
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


def hack_stored_versions(self):
    return self._stored_versions


def hack_stored_versions_setter(self, stored_versions):
    self._stored_versions = stored_versions


def hack_accepted_names(self):
    return self._accepted_names


def hack_accepted_names_setter(self, accepted_names):
    self._accepted_names = accepted_names


def hack_conditions(self):
    return self._conditions


def hack_conditions_setter(self, conditions):
    self._conditions = conditions


def check_crd_type(crd):
    status = False

    try:
        client.apis.ApiextensionsV1beta1Api().read_custom_resource_definition(crd)
        status = True
    except client.rest.ApiException as e:
        if e.status == 404:
            logger.debug(f'CRD type definition not found for {crd}')
        else:
            logger.debug(f'CRD type definition unreadable for {crd}: {e.reason}')

    return status


def check_ingresses():
    status = False

    k8s_v1b1 = client.ExtensionsV1beta1Api(client.ApiClient(client.Configuration()))

    if k8s_v1b1:
        try:
            if ambassador_single_namespace:
                k8s_v1b1.list_namespaced_ingress(ambassador_namespace)
            else:
                k8s_v1b1.list_ingress_for_all_namespaces()
            status = True
        except ApiException as e:
            logger.debug(f'Ingress check got {e.status}')

    return status


def check_ingress_classes():
    status = False

    api_client = client.ApiClient(client.Configuration())

    if api_client:
        try:
            # Sadly, the Kubernetes Python library is not built with forward-compatibility in mind.
            # Since IngressClass is a new resource, it is not discoverable through the python wrapper apis.
            # Here, we extracted (read copy/pasted) a sample call from k8s_v1b1.list_ingress_for_all_namespaces()
            # where we use the rest ApiClient to read ingressclasses.

            path_params = {}
            query_params = []
            header_params = {}

            header_params['Accept'] = api_client. \
                select_header_accept(['application/json',
                                      'application/yaml',
                                      'application/vnd.kubernetes.protobuf',
                                      'application/json;stream=watch',
                                      'application/vnd.kubernetes.protobuf;stream=watch'])

            header_params['Content-Type'] = api_client. \
                select_header_content_type(['*/*'])

            auth_settings = ['BearerToken']

            api_client.call_api('/apis/networking.k8s.io/v1beta1/ingressclasses', 'GET',
                                path_params,
                                query_params,
                                header_params,
                                auth_settings=auth_settings)
            status = True
        except ApiException as e:
            logger.debug(f'IngressClass check got {e.status}')

    return status


def get_api_resources(group, version):
    api_client = client.ApiClient(client.Configuration())

    if api_client:
        try:
            # Sadly, the Kubernetes Python library supports a method equivalent to `kubectl api-versions`
            # but nothing for `kubectl api-resources`.
            # Here, we extracted (read copy/pasted) a sample call from ApisApi().get_api_versions()
            # where we use the rest ApiClient to list api resources specific to a group.

            path_params = {}
            query_params = []
            header_params = {}

            header_params['Accept'] = api_client. \
                select_header_accept(['application/json'])

            auth_settings = ['BearerToken']

            (data) = api_client.call_api(f'/apis/{group}/{version}', 'GET',
                                         path_params,
                                         query_params,
                                         header_params,
                                         auth_settings=auth_settings,
                                         response_type='V1APIResourceList')
            return data[0]
        except ApiException as e:
            logger.error(f'get_api_resources {e.status}')

    return None


def touch_file(touchfile):
    touchpath = Path(ambassador_basedir, touchfile)
    try:
        touchpath.touch()
    except PermissionError as e:
        logger.error(e)


@click.command()
@click.option("--debug", is_flag=True, help="Enable debugging")
def main(debug):
    if debug:
        logger.setLevel(logging.DEBUG)

    found = None
    root_id = None

    cluster_id = os.environ.get('AMBASSADOR_CLUSTER_ID', os.environ.get('AMBASSADOR_SCOUT_ID', None))
    wanted = ambassador_namespace if ambassador_single_namespace else "default"

    # Go ahead and try connecting to Kube.
    v1 = kube_v1()

    # OK. Do the cluster ID dance. If we already have one from the environment,
    # we're good.

    if cluster_id:
        found = "environment"
    else:
        if v1:
            # No ID from the environment, but we can try a lookup using Kube.
            logger.debug("looking up ID for namespace %s" % wanted)

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

        # One way or the other, we need to generate an ID here.
        cluster_url = "d6e_id://%s/%s" % (root_id, ambassador_id)
        logger.debug("cluster ID URL is %s" % cluster_url)

        cluster_id = str(uuid.uuid5(uuid.NAMESPACE_URL, cluster_url)).lower()

    # How about CRDs?

    if v1:
        # We were able to connect to Kube, so let's try to check for missing CRDs too.

        required_crds = [
            (
                '.ambassador_ignore_crds', 'Main CRDs',
                [
                    'authservices.getambassador.io',
                    'mappings.getambassador.io',
                    'modules.getambassador.io',
                    'ratelimitservices.getambassador.io',
                    'tcpmappings.getambassador.io',
                    'tlscontexts.getambassador.io',
                    'tracingservices.getambassador.io'
                ]
            ),
            (
                '.ambassador_ignore_crds_2', 'Resolver CRDs',
                [
                    'consulresolvers.getambassador.io',
                    'kubernetesendpointresolvers.getambassador.io',
                    'kubernetesserviceresolvers.getambassador.io'
                ]
            ),
            (
                '.ambassador_ignore_crds_3', 'Host CRDs',
                [
                    'hosts.getambassador.io'
                ]
            ),
            (
                '.ambassador_ignore_crds_4', 'LogService CRDs',
                [
                    'logservices.getambassador.io'
                ]
            ),
            (
                '.ambassador_ignore_crds_5', 'DevPortal CRDs',
                [
                    'devportals.getambassador.io'
                ]
            )
        ]

        # Flynn would say "Ew.", but we need to patch this till https://github.com/kubernetes-client/python/issues/376
        # and https://github.com/kubernetes-client/gen/issues/52 are fixed \_(0.0)_/
        client.models.V1beta1CustomResourceDefinitionStatus.accepted_names = \
            property(hack_accepted_names, hack_accepted_names_setter)

        client.models.V1beta1CustomResourceDefinitionStatus.conditions = \
            property(hack_conditions, hack_conditions_setter)

        client.models.V1beta1CustomResourceDefinitionStatus.stored_versions = \
            property(hack_stored_versions, hack_stored_versions_setter)

        known_api_resources = []
        api_resources = get_api_resources("getambassador.io", "v2")
        if api_resources:
            known_api_resources = list(map(lambda r: r.name + '.getambassador.io', api_resources.resources))

        for touchfile, description, required in required_crds:
            for crd in required:
                if not crd in known_api_resources:
                    touch_file(touchfile)

                    logger.debug(f'{description} are not available.' +
                                 ' To enable CRD support, configure the Ambassador CRD type definitions and RBAC,' +
                                 ' then restart the Ambassador pod.')
                    # logger.debug(f'touched {touchpath}')

        if not check_ingress_classes():
            touch_file('.ambassador_ignore_ingress_class')

            logger.debug(f'Ambassador does not have permission to read IngressClass resources.' +
                         ' To enable IngressClass support, configure RBAC to allow Ambassador to read IngressClass'
                         ' resources, then restart the Ambassador pod.')

        if not check_ingresses():
            touch_file('.ambassador_ignore_ingress')

            logger.debug(f'Ambassador does not have permission to read Ingress resources.' +
                         ' To enable Ingress support, configure RBAC to allow Ambassador to read Ingress resources,' +
                         ' then restart the Ambassador pod.')

        # Check for our operator's CRD now
        if check_crd_type('ambassadorinstallations.getambassador.io'):
            touch_file('.ambassadorinstallations_ok')
            logger.debug('ambassadorinstallations.getambassador.io CRD available')
        else:
            logger.debug('ambassadorinstallations.getambassador.io CRD not available')

        # Have we been asked to do Knative support?
        if os.environ.get('AMBASSADOR_KNATIVE_SUPPORT', '').lower() == 'true':
            # Yes. Check for their CRD types.

            if check_crd_type('clusteringresses.networking.internal.knative.dev'):
                touch_file('.knative_clusteringress_ok')
                logger.debug('Knative clusteringresses available')
            else:
                logger.debug('Knative clusteringresses not available')

            if check_crd_type('ingresses.networking.internal.knative.dev'):
                touch_file('.knative_ingress_ok')
                logger.debug('Knative ingresses available')
            else:
                logger.debug('Knative ingresses not available')
    else:
        # If we couldn't talk to Kube, log that, but broadly we'll expect our caller
        # to DTRT around CRDs.

        logger.debug('Kubernetes is not available, so not doing CRD check')

    # Finally, spit out the cluster ID for our caller.
    logger.debug("cluster ID is %s (from %s)" % (cluster_id, found))

    print(cluster_id)


if __name__ == "__main__":
    main()
