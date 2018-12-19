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
import json
import logging
import os
import shutil
import signal
import threading
import time
import uuid
import yaml

from urllib3.exceptions import ProtocolError
from typing import Optional, Dict

from kubernetes import watch
from kubernetes.client.rest import ApiException
from ambassador import Config, Scout
from ambassador.utils import kube_v1, read_cert_secret, save_cert, TLSPaths
from ambassador.ir import IR
from ambassador.envoy import V2Config

from ambassador.VERSION import Version

__version__ = Version
ambassador_id = os.getenv("AMBASSADOR_ID", "default")

logging.basicConfig(
    level=logging.INFO,  # if appDebug else logging.INFO,
    format="%%(asctime)s kubewatch [%%(process)d T%%(threadName)s] %s %%(levelname)s: %%(message)s" % __version__,
    datefmt="%Y-%m-%d %H:%M:%S"
)

# logging.getLogger("datawire.scout").setLevel(logging.DEBUG)
logger = logging.getLogger("kubewatch")
logger.setLevel(logging.INFO)

KEY = "getambassador.io/config"


def is_annotated(svc):
    annotations = svc.metadata.annotations
    return annotations and KEY in annotations


def get_annotation(svc):
    return svc.metadata.annotations[KEY] if is_annotated(svc) else None


def get_source(svc):
    return "service %s, namespace %s" % (svc.metadata.name, svc.metadata.namespace)


def get_filename(svc):
    return "%s-%s.yaml" % (svc.metadata.name, svc.metadata.namespace)


class Restarter(threading.Thread):

    def __init__(self, ambassador_config_dir, namespace, envoy_config_file, delay, pid):
        threading.Thread.__init__(self, daemon=True, name="Restarter")

        self.ambassador_config_dir = ambassador_config_dir
        self.config_root = os.path.abspath(os.path.dirname(self.ambassador_config_dir))
        self.envoy_config_dir = os.path.join(self.config_root, "envoy")
        self.namespace = namespace
        self.envoy_config_file = envoy_config_file
        self.delay = delay
        self.pid = pid
        self.cluster_id = None

        self.mutex = threading.Condition()
        # This holds how many times we have been poked.
        self.pokes = 0
        # This holds how many pokes we have actually processed.
        self.processed = self.pokes
        self.restart_count = 0

        self.configs = {}

        self.last_bootstrap = None

        # Read the base configuration...
        self.read_fs(self.ambassador_config_dir)

        # ...then pull in anything updated by the restarter logic.
        while True:
            if not os.path.exists("%s-%s" % (self.ambassador_config_dir, self.restart_count + 1)):
                break
            else:
                self.restart_count += 1

        path = "%s-%s" % (self.ambassador_config_dir, self.restart_count)
        self.read_fs(path)

    def set_cluster_id(self, cluster_id) -> None:
        with self.mutex:
            self.cluster_id = cluster_id

    def tls_secret_resolver(self, secret_name: str, context: str, cert_dir=None) -> Optional[Dict[str, str]]:
        (cert, key, data) = read_cert_secret(kube_v1(), secret_name, self.namespace)
        if not (cert and key):
            return None

        certificate_chain_path = ""
        private_key_path = ""

        if context == 'server':
            cert_dir = TLSPaths.cert_dir.value
            certificate_chain_path = TLSPaths.tls_crt.value
            private_key_path = TLSPaths.tls_key.value
        elif context == 'client':
            # TODO
            pass
        else:
            if cert_dir is None:
                cert_dir = os.path.join("/ambassador/", context)

            cert_paths = TLSPaths.generate(cert_dir)
            certificate_chain_path = cert_paths['crt']
            private_key_path = cert_paths['key']

        logger.debug("saving contents of secret %s to %s for context %s" % (secret_name, cert_dir, context))
        save_cert(cert, key, cert_dir)

        return {
            'certificate_chain_file': certificate_chain_path,
            'private_key_file': private_key_path
        }

    def read_fs(self, path):
        if os.path.exists(path):
            logger.debug("Merging config inputs from %s" % path)

            for name in os.listdir(path):
                if name.endswith(".yaml"):
                    filepath = os.path.join(path, name)

                    with open(filepath) as fd:
                        self.configs[name] = self.read_yaml(fd.read(), "file %s" % filepath)

                    logger.debug("Loaded %s" % filepath)

    def run(self):
        while True:
            logger.debug("Restarter sleeping...")
            # This sleep rate limits the number of restart attempts.
            time.sleep(self.delay)

            with self.mutex:
                changes = self.pokes - self.processed

                if changes > 0:
                    logger.debug("Processing %s changes" % changes)

                    try:
                        self.restart(changes)
                    except Exception as e:
                        logging.exception("could not restart Envoy: %s" % e)

                    self.processed += changes
                else:
                    logger.debug("No changes, cycling")

    @staticmethod
    def safe_write(temp_dir, target_dir, target_name, serialized):
        temp_path = "%s-%s" % (temp_dir, target_name)

        with open(temp_path, "w") as o:
            o.write(serialized)
            o.write("\n")

        target_path = os.path.abspath(os.path.join(target_dir, target_name))

        os.rename(temp_path, target_path)

        return target_path

    def restart(self, changes=None):
        if changes is None:
            with self.mutex:
                changes = self.pokes - self.processed

        self.restart_count += 1
        output = "%s-%s" % (self.ambassador_config_dir, self.restart_count)
        bootstrap_config, ads_config = self.generate_config(changes, output)

        bootstrap_serialized = json.dumps(bootstrap_config, sort_keys=True, indent=4)
        need_restart = False
        rewrite_bootstrap = False

        if not self.last_bootstrap:
            rewrite_bootstrap = True
        elif bootstrap_serialized != self.last_bootstrap:
            need_restart = True
            rewrite_bootstrap = True

        self.last_bootstrap = bootstrap_serialized

        if rewrite_bootstrap:
            bootstrap_path = self.safe_write(output, self.config_root, "bootstrap-ads.json",
                                             bootstrap_serialized)

            logger.debug("Rewrote bootstrap to %s" % bootstrap_path)

        envoy_path = self.safe_write(output, self.envoy_config_dir, "envoy.json",
                                     json.dumps(ads_config, sort_keys=True, indent=4))

        logger.debug("Wrote configuration %d to %s" % (self.restart_count, envoy_path))

        if need_restart:
            logger.warning("RESTART REQUIRED: bootstrap changed")

            with open(os.path.join(self.config_root, "notices.json"), "w") as notices:
                notices.write(json.dumps([{ 'level': 'WARNING',
                                            'message': 'RESTART REQUIRED! after bootstrap change' }],
                                         sort_keys=True, indent=4))
                notices.write("\n")

        if self.pid:
            os.kill(self.pid, signal.SIGHUP)

    def generate_config(self, changes, output):
        if os.path.exists(output):
            shutil.rmtree(output)
        os.makedirs(output)
        for filename, config in self.configs.items():
            path = os.path.join(output, filename)
            with open(path, "w") as fd:
                fd.write(config)
            logger.debug("Wrote %s to %s" % (filename, path))

        plural = "" if (changes == 1) else "s"

        logger.info("generating config with gencount %d (%d change%s)" %
                    (self.restart_count, changes, plural))

        aconf = Config()
        aconf.load_from_directory(output)
        ir = IR(aconf, tls_secret_resolver=self.tls_secret_resolver)
        envoy_config = V2Config(ir)

        ads_config = {
            '@type': '/envoy.config.bootstrap.v2.Bootstrap',
            'static_resources': envoy_config.static_resources
        }

        bootstrap_config = dict(envoy_config.bootstrap)

        scout = Scout(install_id=self.cluster_id)
        scout_args = { "gencount": self.restart_count }

        if not os.environ.get("AMBASSADOR_DISABLE_FEATURES", None):
            scout_args["features"] = ir.features()

        result = scout.report(mode="kubewatch", action="reconfigure", **scout_args)
        notices = result.pop("notices", [])

        logger.debug("scout result %s" % json.dumps(result, sort_keys=True, indent=4))

        for notice in notices:
            logger.log(logging.getLevelName(notice.get('level', 'WARNING')), notice.get('message', '?????'))

        return bootstrap_config, ads_config

    def update_from_service(self, svc):
        key = get_filename(svc)
        source = get_source(svc)
        config = get_annotation(svc)

        logger.debug("update_from_svc: key %s, config %s" % (key, yaml.safe_dump(config)))

        if config is None:
            self.delete(svc)
        else:
            self.update(key, self.read_yaml(config, source))

    @staticmethod
    def read_yaml(raw_yaml, source):
        metadata = "\n".join([
            '---',
            'apiVersion: v0.1',
            'kind: Pragma',
            'ambassador_id: %s' % ambassador_id,
            'source: "%s"' % source,
            'autogenerated: true'
        ])

        sep = "---\n" if not raw_yaml.lstrip().startswith("---") else ""

        return metadata + "\n" + sep + raw_yaml

    def update(self, key, config):
        logger.debug("update: including key %s" % key)

        with self.mutex:
            need_update = False

            if key not in self.configs:
                need_update = True
            elif config != self.configs[key]:
                need_update = True

            if need_update:
                self.configs[key] = config
                self.poke()

    def delete(self, svc):
        with self.mutex:
            key = get_filename(svc)

            logger.debug("delete: dropping key %s" % key)

            if key in self.configs:
                del self.configs[key]
                self.poke()

    def poke(self):
        with self.mutex:
            if self.processed == self.pokes:
                logger.debug("Scheduling restart")
            self.pokes += 1


class KubeWatcher:
    def __init__(self, logger, restarter):
        self.cluster_id = os.environ.get('AMBASSADOR_CLUSTER_ID',
                                         os.environ.get('AMBASSADOR_SCOUT_ID', None))
        self.need_sync = True
        self.restarter_started = False

        self.logger = logger

        self.restarter = restarter
        self.namespace = self.restarter.namespace
        self.single_namespace = bool("AMBASSADOR_SINGLE_NAMESPACE" in os.environ)

        if self.cluster_id:
            self.logger.info("starting with ID %s" % self.cluster_id)

        self.logger.info("namespace %s, watching %s" %
                         (self.namespace,
                          "just this namespace" if self.single_namespace else "all namespaces"))

    def run(self, sync_only=False):
        self.logger.debug("starting run")

        while True:
            # Catch exceptions... just in case.
            try:
                # Try for a Kube connection.
                v1 = kube_v1()

                if v1:
                    self.logger.debug("connected to Kubernetes!")

                    # Manage cluster_id if needed...
                    if not self.cluster_id:
                        # Nope. Try to figure it out.
                        self.get_cluster_id(v1)

                    # ...then do sync if needed.
                    if self.need_sync:
                        self.sync(v1)

                # Whether or not we got a Kube connection, generate the initial Envoy config if needed
                # (including setting the restarter's cluster ID).
                if self.need_sync:
                    logger.debug("Generating initial Envoy config")

                    self.restarter.set_cluster_id(self.cluster_id)
                    self.restarter.restart()

                    self.need_sync = False

                # If we're just doing the sync, dump the cluster_id to stdout and then bail.
                if sync_only:
                    print(self.cluster_id)
                    break

                # We're not just doing a resync. Start the restarter loop if we need to.
                if not self.restarter_started:
                    self.restarter.start()
                    self.restarter_started = True

                # Finally, start watching, if we need to.
                if v1:
                    self.watch(v1)

            except KeyboardInterrupt:
                # If the user hit ^C, this is an interactive session and we should bail.
                self.logger.warning("Exiting on ^C")
                raise

            except (ProtocolError, ApiException) as e:
                # If any Kubernetes thing failed, cycle (unless told otherwise)
                self.logger.warning("Kubernetes access failure! %s" % e)

                if 'AMBASSADOR_NO_KUBEWATCH_RETRY' in os.environ:
                    logger.info("not restarting! AMBASSADOR_NO_KUBEWATCH_RETRY is set")
                    raise

            except Exception:
                # WTFO.
                self.logger.warning("kubewatch failed!")
                raise

            finally:
                # If we're cycling, wait 10 seconds.
                logger.debug("10-second watch loop delay")
                time.sleep(10)

    def get_cluster_id(self, v1):
        wanted = self.namespace if self.single_namespace else "default"
        found = None
        root_id = None

        self.logger.debug("looking up ID for namespace %s" % wanted)

        try:
            ret = v1.read_namespace(wanted)
            root_id = ret.metadata.uid
            found = "namespace %s" % wanted
        except ApiException as e:
            # This means our namespace wasn't found?
            self.logger.error("couldn't read namespace %s? %s" %
                              (self.namespace, e))

        if not root_id:
            # OK, so we had a crack at this and something went wrong. Give up and hardcode
            # something.
            root_id = "00000000-0000-0000-0000-000000000000"
            found = "hardcoded ID"

        cluster_url = "d6e_id://%s/%s" % (root_id, ambassador_id)
        self.logger.debug("cluster ID URL is %s" % cluster_url)

        self.cluster_id = str(uuid.uuid5(uuid.NAMESPACE_URL, cluster_url)).lower()
        self.logger.info("cluster ID is %s (from %s)" % (self.cluster_id, found))

    def sync(self, v1):
        # We have a Kube API! Check for annotations and such.
        svc_list = None

        self.logger.debug("sync attempting to list services for %s" %
                          (("namespace %s" % self.namespace) if self.single_namespace else "all namespaces"))

        if self.single_namespace:
            svc_list = v1.list_namespaced_service(self.namespace)
        else:
            svc_list = v1.list_service_for_all_namespaces()

        if svc_list:
            self.logger.debug("sync found %d service%s" %
                         (len(svc_list.items), ("" if (len(svc_list.items) == 1) else "s")))

            for svc in svc_list.items:
                self.restarter.update_from_service(svc)
        else:
            self.logger.debug("sync found no services")


    def watch(self, v1):
        w = watch.Watch()

        watched = None

        self.logger.debug("watch attempting to watch services for %s" %
                          (("namespace %s" % self.namespace) if self.single_namespace else "all namespaces"))

        if self.single_namespace:
            watched = w.stream(v1.list_namespaced_service, namespace=self.namespace)
        else:
            watched = w.stream(v1.list_service_for_all_namespaces)

        for evt in watched:
            self.logger.debug("event: %s %s/%s" %
                              (evt["type"],
                               evt["object"].metadata.namespace, evt["object"].metadata.name))
            # sys.stdout.flush()

            if evt["type"] == "DELETED":
                self.restarter.delete(evt["object"])
            else:
                self.restarter.update_from_service(evt["object"])

        # If here, something strange happened and the watch loop exited on its own.
        # Let our caller handle that.
        logger.info("watch loop exited?")


@click.command()
@click.argument("mode", type=click.Choice(["sync", "watch"]))
@click.argument("ambassador_config_dir")
@click.argument("envoy_config_file")
@click.option("--debug", is_flag=True,
              help="Enable debugging")
@click.option("-d", "--delay", type=click.FLOAT, default=1.0,
              help="The minimum delay in seconds between restart attempts.")
@click.option("-p", "--pid", type=click.INT,
              help="The pid to kill with SIGHUP in order to iniate a restart.")
def main(mode, ambassador_config_dir, envoy_config_file, debug, delay, pid):
    """This script watches the kubernetes API for changes in services. It
    collects ambassador configuration imput from the ambassador
    annotation on any services, and whenever these change, it will
    generate a new set of ambassador configuration inputs. It will
    then diff these inputs with the previous configuration and if
    necessary regenerate an envoy configuration and initiate a reload
    of envoy configuration by signaling the ambex PID.

    This script will rate limit attempts to reconfigure envoy based on
    the supplied `--delay` parameter:

      --delay (a parameter of this script)
    
         This is the reconfig delay. It limits the minimum time this
         script will allow between subsequent reconfigure attempts.
    """

    if debug:
        logger.setLevel(logging.DEBUG)

    namespace = os.environ.get('AMBASSADOR_NAMESPACE', 'default')

    restarter = Restarter(ambassador_config_dir, namespace, envoy_config_file, delay, pid)
    watcher =  KubeWatcher(logger, restarter)

    if mode == "sync":
        watcher.run(sync_only=True)
    elif mode == "watch":
        watcher.run(sync_only=False)
    else:
        raise ValueError(mode)

if __name__ == "__main__":
    main()
