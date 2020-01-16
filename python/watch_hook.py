#!/usr/bin/python

from ambassador.utils import ParsedService as Service

from typing import Dict, Optional, Tuple, TYPE_CHECKING

import sys

import json
import logging
import os

from ambassador import Config, IR
from ambassador.config.resourcefetcher import ResourceFetcher
from ambassador.utils import SecretInfo, SavedSecret, SecretHandler

if TYPE_CHECKING:
    from ambassador.ir.irresource import IRResource

# default AES's Secret name
# (by default, we assume it will be in the same namespace as Ambassador)
DEFAULT_AES_SECRET_NAME = "ambassador-edge-stack"

# the name of some env vars that can be used for overriding
# the AES's Secret name/namespace
ENV_AES_SECRET_NAME = "AMBASSADOR_AES_SECRET_NAME"
ENV_AES_SECRET_NAMESPACE = "AMBASSADOR_AES_SECRET_NAMESPACE"


# Fake SecretHandler for our fake IR, below.

class SecretRecorder(SecretHandler):
    def __init__(self, logger: logging.Logger) -> None:
        super().__init__(logger, "-source_root-", "-cache_dir-", "0")
        self.needed: Dict[Tuple[str, str], SecretInfo] = {}

    # Record what was requested, and always return success.
    def load_secret(self, resource: 'IRResource',
                    secret_name: str, namespace: str) -> Optional[SecretInfo]:
        self.logger.debug("SecretRecorder (%s %s): load secret %s in namespace %s" %
                          (resource.kind, resource.name, secret_name, namespace))

        return self.record_secret(secret_name, namespace)

    def record_secret(self, secret_name: str, namespace: str) -> Optional[SecretInfo]:
        secret_key = (secret_name, namespace)

        if secret_key not in self.needed:
            self.needed[secret_key] = SecretInfo(secret_name, namespace, 'needed-secret', '-crt-', '-key-',
                                                 decode_b64=False)
        return self.needed[secret_key]

    # Secrets that're still needed also get recorded.
    def still_needed(self, resource: 'IRResource', secret_name: str, namespace: str) -> None:
        self.logger.debug("SecretRecorder (%s %s): secret %s in namespace %s is still needed" %
                          (resource.kind, resource.name, secret_name, namespace))

        self.record_secret(secret_name, namespace)

    # Never cache anything.
    def cache_secret(self, resource: 'IRResource', secret_info: SecretInfo):
        self.logger.debug("SecretRecorder (%s %s): skipping cache step for secret %s in namespace %s" %
                          (resource.kind, resource.name, secret_info.name, secret_info.namespace))
                          
        return SavedSecret(secret_info.name, secret_info.namespace, '-crt-path-', '-key-path-', '-user-path-',
                           { 'tls.crt': '-crt-', 'tls.key': '-key-', 'user.key': '-user-' })


# XXX Sooooo there's some ugly stuff here.
#
# We need to do a little bit of the same work that the IR does for things like
# managing Resolvers and parsing service names. However, we really don't want to
# do all the work of instantiating an IR.
#
# The solution here is to subclass the IR and take advantage of the watch_only
# initialization keyword, which skips the hard parts of building an IR.

class FakeIR(IR):
    def __init__(self, aconf: Config, logger=None) -> None:
        # If we're asked about a secret, record interest in that secret.
        self.secret_recorder = SecretRecorder(logger)

        # If we're asked about a file, it's good.
        file_checker = lambda path: True

        super().__init__(aconf, logger=logger, watch_only=True,
                         secret_handler=self.secret_recorder, file_checker=file_checker)

    # Don't bother actually saving resources that come up when working with
    # the faked modules.
    def save_resource(self, resource: 'IRResource') -> 'IRResource':
        return resource


class WatchHook:
    def __init__(self, logger, yaml_stream) -> None:
        # Watch management

        self.logger = logger

        self.consul_watches = []
        self.kube_watches = []

        self.load_yaml(yaml_stream)

    def add_kube_watch(self, what: str, kind: str, namespace: Optional[str],
                       field_selector: Optional[str]=None, label_selector: Optional[str]=None) -> None:
        watch = { "kind": kind }

        if namespace:
            watch["namespace"] = namespace

        if field_selector:
            watch["field-selector"] = field_selector

        if label_selector:
            watch["label-selector"] = label_selector

        self.logger.debug(f"{what}: add watch {watch}")
        self.kube_watches.append(watch)

    def load_yaml(self, yaml_stream):
        self.aconf = Config()

        fetcher = ResourceFetcher(self.logger, self.aconf, watch_only=True)
        fetcher.parse_watt(yaml_stream.read())

        self.aconf.load_all(fetcher.sorted())

        # We can lift mappings straight from the aconf...
        mappings = self.aconf.get_config('mappings') or {}

        # ...but we need the fake IR to deal with resolvers and TLS contexts.
        self.fake = FakeIR(self.aconf, logger=self.logger)

        self.logger.debug("IR: %s" % self.fake.as_json())

        resolvers = self.fake.resolvers
        contexts = self.fake.tls_contexts

        self.logger.debug(f'mappings: {len(mappings)}')
        self.logger.debug(f'resolvers: {len(resolvers)}')
        self.logger.debug(f'contexts: {len(contexts)}')

        global_resolver = self.fake.ambassador_module.get('resolver', None)

        global_label_selector = os.environ.get('AMBASSADOR_LABEL_SELECTOR', '')
        self.logger.debug('label-selector: %s' % global_label_selector)

        # watch the AES Secret if the edge stack is running
        if self.fake.edge_stack_allowed:
            aes_secret_name = os.getenv(ENV_AES_SECRET_NAME, DEFAULT_AES_SECRET_NAME)
            aes_secret_namespace = os.getenv(ENV_AES_SECRET_NAMESPACE, Config.ambassador_namespace)
            self.logger.debug(f'edge stack detected: need secret {aes_secret_name}.{aes_secret_namespace}')
            self.add_kube_watch(f'Secret {aes_secret_name}', 'secret', namespace=aes_secret_namespace,
                                field_selector=f"metadata.name={aes_secret_name}")

        # Walk hosts.
        for host in self.fake.get_hosts():
            sel = host.get('selector') or {}
            match_labels = sel.get('matchLabels') or {}

            label_selector = None

            if match_labels:
                label_selector = ','.join([f"{l}={v}" for l, v in match_labels.items()])

            for wanted_kind in ['service', 'secret']:
                self.add_kube_watch(f"Host {host.name}", wanted_kind, host.namespace,
                                    label_selector=label_selector)

        for mname, mapping in mappings.items():
            res_name = mapping.get('resolver', None)
            res_source = 'mapping'

            if not res_name:
                res_name = global_resolver
                res_source = 'defaults'

            ctx_name = mapping.get('tls', None)

            self.logger.debug(
                f'Mapping {mname}: resolver {res_name} from {res_source}, service {mapping.service}, tls {ctx_name}')

            if res_name:
                resolver = resolvers.get(res_name, None)
                self.logger.debug(f'-> resolver {resolver}')

                if resolver:
                    svc = Service(logger, mapping.service, ctx_name)

                    if resolver.kind == 'ConsulResolver':
                        self.logger.debug(f'Mapping {mname} uses Consul resolver {res_name}')

                        # At the moment, we stuff the resolver's datacenter into the association
                        # ID for this watch. The ResourceFetcher relies on that.

                        self.consul_watches.append(
                            {
                                "id": resolver.datacenter,
                                "consul-address": resolver.address,
                                "datacenter": resolver.datacenter,
                                "service-name": svc.hostname
                            }
                        )
                    elif resolver.kind == 'KubernetesEndpointResolver':
                        host = svc.hostname
                        namespace = Config.ambassador_namespace

                        if not host:
                            # This is really kind of impossible.
                            self.logger.error(f"KubernetesEndpointResolver {res_name} has no 'hostname'")
                            continue

                        if "." in host:
                            (host, namespace) = host.split(".", 2)[0:2]

                        self.logger.debug(f'...kube endpoints: svc {svc.hostname} -> host {host} namespace {namespace}')

                        self.add_kube_watch(f"endpoint", "endpoints", namespace,
                                            label_selector=global_label_selector,
                                            field_selector=f"metadata.name={host}")

        for secret_key, secret_info in self.fake.secret_recorder.needed.items():
            self.logger.debug(f'need secret {secret_info.name}.{secret_info.namespace}')

            self.add_kube_watch(f"needed secret", "secret", secret_info.namespace,
                                field_selector=f"metadata.name={secret_info.name}")

        if self.fake.edge_stack_allowed:
            # If the edge stack is allowed, make sure we watch for our fallback context.
            self.add_kube_watch("Fallback TLSContext", "TLSContext", namespace=Config.ambassador_namespace)

        ambassador_knative_requested = (os.environ.get("AMBASSADOR_KNATIVE_SUPPORT", "-unset-").lower() == 'true')

        if ambassador_knative_requested:
            self.logger.debug('Looking for Knative support...')

            ambassador_basedir = os.environ.get('AMBASSADOR_CONFIG_BASE_DIR', '/ambassador')

            if os.path.exists(os.path.join(ambassador_basedir, '.knative_clusteringress_ok')):
                # Watch for clusteringresses.networking.internal.knative.dev in any namespace and with any labels.

                self.logger.debug('watching for clusteringresses.networking.internal.knative.dev')
                self.add_kube_watch("Knative clusteringresses", "clusteringresses.networking.internal.knative.dev",
                                    None)

            if os.path.exists(os.path.join(ambassador_basedir, '.knative_ingress_ok')):
                # Watch for ingresses.networking.internal.knative.dev in any namespace and
                # with any labels.

                self.add_kube_watch("Knative ingresses", "ingresses.networking.internal.knative.dev", None)

        self.watchset = {
            "kubernetes-watches": self.kube_watches,
            "consul-watches": self.consul_watches
        }

        save_dir = os.environ.get('AMBASSADOR_WATCH_DIR', '/tmp')

        if save_dir:
            json.dump(self.watchset, open(os.path.join(save_dir, 'watch.json'), "w"))


#### Mainline.

if __name__ == "__main__":
    loglevel = logging.INFO

    args = sys.argv[1:]

    if args:
        if args[0] == '--debug':
            loglevel = logging.DEBUG
            args.pop(0)
        elif args[0].startswith('--'):
            raise Exception(f'Usage: {os.path.basename(sys.argv[0])} [--debug] [path]')

    logging.basicConfig(
        level=loglevel,
        format="%(asctime)s watch-hook %(levelname)s: %(message)s",
        datefmt="%Y-%m-%d %H:%M:%S"
    )

    alogger = logging.getLogger('ambassador')
    alogger.setLevel(logging.INFO)

    logger = logging.getLogger('watch_hook')
    logger.setLevel(loglevel)

    yaml_stream = sys.stdin

    if args:
        yaml_stream = open(args[0], "r")

    wh = WatchHook(logger, yaml_stream)

    json.dump(wh.watchset, sys.stdout)

