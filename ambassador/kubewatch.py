import sys

import binascii
import click
import json
import os
import shutil
import signal
import subprocess
import threading
import time
import traceback

import yaml

from kubernetes import client, config, watch
from ambassador.config import Config

KEY = "getambassador.io/config"

def is_annotated(svc):
    annotations = svc.metadata.annotations
    return annotations and KEY in annotations

def get_annotation(svc):
    return svc.metadata.annotations[KEY] if is_annotated(svc) else None

def get_filename(svc):
    return "%s.yaml" % svc.metadata.name

class Restarter(threading.Thread):

    def __init__(self, ambassador_config_dir, namespace, envoy_config_file, delay, pid):
        threading.Thread.__init__(self, daemon=True)

        self.ambassador_config_dir = ambassador_config_dir
        self.namespace = namespace
        self.envoy_config_file = envoy_config_file
        self.delay = delay
        self.pid = pid

        self.mutex = threading.Condition()
        # This holds how many times we have been poked.
        self.pokes = 0
        # This holds how many pokes we have actually processed.
        self.processed = self.pokes
        self.restart_count = 0

        self.configs = {}

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

    def read_fs(self, path):
        if os.path.exists(path):
            print("Merging config inputs from %s" % path)

            for name in os.listdir(path):
                if name.endswith(".yaml"):
                    with open(os.path.join(path, name)) as fd:
                        self.configs[name] = fd.read()

                    print ("Loaded %s" % os.path.join(path, name))

    def changes(self):
        with self.mutex:
            delta = self.pokes - self.processed
        return delta

    def run(self):
        while True:
            # This sleep rate limits the number of restart attempts.
            time.sleep(self.delay)
            with self.mutex:
                changes = self.changes()
                if changes > 0:
                    print("Processing %s changes" % (changes))
                    try:
                        self.restart()
                    except:
                        traceback.print_exc()
                    self.processed += changes

    def restart(self):
        self.restart_count += 1
        output = "%s-%s" % (self.ambassador_config_dir, self.restart_count)
        config = self.generate_config(output)
        base, ext = os.path.splitext(self.envoy_config_file)
        target = "%s-%s%s" % (base, self.restart_count, ext)
        os.rename(config, target)
        print ("Moved valid configuration %s to %s" % (config, target))
        if self.pid:
            os.kill(self.pid, signal.SIGHUP)

    def generate_config(self, output):
        if os.path.exists(output):
            shutil.rmtree(output)
        os.makedirs(output)
        for filename, config in self.configs.items():
            path = os.path.join(output, filename)
            with open(path, "w") as fd:
                fd.write(config)
            print ("Wrote %s to %s" % (filename, path))

        aconf = Config(output)
        rc = aconf.generate_envoy_config(mode="kubewatch")

        print("Scout reports %s" % json.dumps(rc.scout_result))       

        if rc:
            envoy_config = "%s-%s" % (output, "envoy.json")
            aconf.pretty(rc.envoy_config, out=open(envoy_config, "w"))
            try:
                result = subprocess.check_output(["/usr/local/bin/envoy", "--base-id", "1", "--mode", "validate",
                                                  "-c", envoy_config])
                if result.strip().endswith(b" OK"):
                    print ("Configuration %s valid" % envoy_config)
                    return envoy_config
            except subprocess.CalledProcessError:
                print ("Invalid envoy config")
                with open(envoy_config) as fd:
                    print(fd.read())
        else:
            print("Could not generate new Envoy configuration: %s" % rc.error)
            print("Raw template output:")
            print("%s" % rc.raw)

        raise ValueError("Unable to generate config")

    def update_from_service(self, svc):
        config = get_annotation(svc)
        if config is None:
            self.delete(svc)
        else:
            key = get_filename(svc)
            self.update(key, config)

    def update(self, key, config):
        print("update: including key %s" % key)

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
            if key in self.configs:
                del self.configs[key]
                self.poke()

    def poke(self):
        with self.mutex:
            if self.processed == self.pokes:
                print ("Scheduling restart")
            self.pokes += 1


def kube_v1():
    # Assume we got nothin'.
    k8s_api = None

    # XXX: is there a better way to check if we are inside a cluster or not?
    if "KUBERNETES_SERVICE_HOST" in os.environ:
        # If this goes horribly wrong and raises an exception (it shouldn't),
        # we'll crash, and Kubernetes will kill the pod. That's probably not an
        # unreasonable response.
        config.load_incluster_config()
        k8s_api = client.CoreV1Api()
    else:
        # Here, we might be running in docker, in which case we'll likely not
        # have any Kube secrets, and that's OK.
        try:
            config.load_kube_config()
            k8s_api = client.CoreV1Api()
        except FileNotFoundError:
            # Meh, just ride through.
            print("No K8s")
            pass

    return k8s_api

def check_cert_file(path):
    readable = False

    try:
        data = open(path, "r").read()

        if data and (len(data) > 0):
            readable = True
    except OSError:
        pass
    except IOError:
        pass

    return readable

def read_cert_secret(k8s_api, secret_name, namespace):
    cert_data = None
    cert = None
    key = None

    try:
        cert_data = k8s_api.read_namespaced_secret(secret_name, namespace)
    except client.rest.ApiException as e:
        if e.reason == "Not Found":
            pass
        else:
            print("secret %s/%s could not be read: %s" % (namespace, secret_name, e))

    if cert_data and cert_data.data:
        cert_data = cert_data.data
        cert = cert_data.get('tls.crt', None)

        if cert:
            cert = binascii.a2b_base64(cert)

        key = cert_data.get('tls.key', None)

        if key:
            key = binascii.a2b_base64(key)

    return (cert, key, cert_data)

def save_cert(cert, key, dir):
    try:
        os.makedirs(dir)
    except FileExistsError:
        pass

    open(os.path.join(dir, "tls.crt"), "w").write(cert.decode("utf-8"))
    open(os.path.join(dir, "tls.key"), "w").write(key.decode("utf-8"))

def sync(restarter):
    v1 = kube_v1()

    if v1:
        # We have a Kube API! Do we have an ambassador-config ConfigMap?
        cm_names = [ x.metadata.name 
                     for x in v1.list_namespaced_config_map(restarter.namespace).items ]

        if 'ambassador-config' in cm_names:
            config_data = v1.read_namespaced_config_map("ambassador-config", restarter.namespace)

            if config_data:
                for key, config_yaml in config_data.data.items():
                    # print("ambassador-config: found %s" % key)
                    restarter.update(key, config_yaml)

        # If we don't already see a TLS server key in its usual spot...
        if not check_cert_file("/etc/certs/tls.crt"):
            # ...then try pulling keys directly from the configmaps.
            (server_cert, server_key, server_data) = read_cert_secret(v1, "ambassador-certs", 
                                                                      restarter.namespace)
            (client_cert, _, client_data) = read_cert_secret(v1, "ambassador-cacert", 
                                                             restarter.namespace)

            if server_cert and server_key:
                tls_mod = {
                    "apiVersion": "ambassador/v0",
                    "kind": "Module",
                    "name": "tls",
                    "config": {
                        "server": {
                            "enabled": True,
                            "cert_chain_file": "/etc/certs/tls.crt",
                            "private_key_file": "/etc/certs/tls.key"
                        }
                    }
                }

                save_cert(server_cert, server_key, "/etc/certs")

                if client_cert:
                    tls_mod['config']['client'] = {
                        "enabled": True,
                        "cacert_chain_file": "/etc/cacert/tls.pem"
                    }

                    if client_data.get('cert_required', None):
                        tls_mod['config']['client']["cert_required"] = True

                    save_cert(client_cert, None, "/etc/cacert")

                restarter.update("tls.yaml", yaml.safe_dump(tls_mod))

        # Next, check for annotations and such.
        for svc in v1.list_namespaced_service(restarter.namespace).items:
            restarter.update_from_service(svc)

    print ("Changes detected, regenerating envoy config.")
    restarter.restart()

def watch_loop(restarter):
    v1 = kube_v1()

    if v1:
        w = watch.Watch()
        for evt in w.stream(v1.list_service_for_all_namespaces):
            print("Event: %s %s/%s" % (evt["type"], 
                                       evt["object"].metadata.namespace, evt["object"].metadata.name))
            sys.stdout.flush()

            if evt["type"] == "DELETED":
                restarter.delete(evt["object"])
            else:
                restarter.update_from_service(evt["object"])
    else:
        print("No K8s, idling")
        while True:
            time.sleep(60)

@click.command()
@click.argument("mode", type=click.Choice(["sync", "watch"]))
@click.argument("ambassador_config_dir")
@click.argument("envoy_config_file")
@click.option("-d", "--delay", type=click.FLOAT, default=5.0,
              help="The minimum delay in seconds between restart attempts.")
@click.option("-p", "--pid", type=click.INT,
              help="The pid to kill with SIGHUP in order to iniate a restart.")
def main(mode, ambassador_config_dir, envoy_config_file, delay, pid):
    """This script watches the kubernetes API for changes in services. It
    collects ambassador configuration imput from the ambassador
    annotation on any services, and whenever these change, it will
    generate a new set of ambassador configuration inputs. It will
    then diff these inputs with the previous configuration and if
    necessary regenerate an envoy configuration and initiate an envoy
    restart.

    Envoy is engineered to restart with zero connection loss, but this
    process takes time and needs to be properly managed in two ways:
    both timing of restarts and validation of inputs to the new envoy.

    Restarting with zero connection loss takes time. The new envoy
    initiates a drain period in the old envoy (see envoy's
    --drain-time-s option), and then there is a further delay before
    the new envoy shuts down the old one (see envoy's
    --parent-shutdown-time-s option). Any attempt to initiate another
    restart while the previous restart is already in progress will
    fail. This means we need to take further care not to initiate
    restarts too frequently. This leaves us with three delay
    parameters that needed to be tuned with increasing values that
    have sufficient margins:
    
      --drain-time-s (an envoy parameter)
    
         This is time permitted for active connections to drain from
         the old envoy. This is the smallest value. What you want to
         tune this to depends on your scenario, e.g. edge scenarios
         will likely want to permit more drain time, maybe 5 or 10
         minutes.
    
      --parent-shutdown-time-s (an envoy parameter)
    
         This is the time the new envoy gives the old envoy to
         complete it's drain before shutting it down. This is an
         absolute time measured from the initiation of the restart,
         and so it doesn't make sense for it to be less than the
         configured drain time. It should also be a bit larger than
         the drain time to account for timing discrepencies. Envoy
         examples seem to set it to 50% more than the drain time.
    
      --delay (a parameter of this script)
    
         This is the restart delay. It limits the minimum time this
         script will allow between subsequent restarts. This should be
         configured to be larger than the --parent-shutdown-time-s
         option by a reasonable margin.
    
    In addition to the timing involved, envoy's restart machinery will
    die completely (both killing the old and new envoy) if the new
    envoy is supplied with an invalid configuration. This script takes
    care to ensure that all inputs are fully validated using envoy's
    --mode validate option in order to ensure that we never attempt to
    restart with an invalid configuration. It also keeps a full
    history of all configurations along with the errors from any
    invalid configurations to aid in debugging if invalid
    configuration inputs are supplied in any annotations, or if there
    is an ambassador bug encountered when processing an annotation.

    """

    namespace = os.environ.get('AMBASSADOR_NAMESPACE', 'default')

    restarter = Restarter(ambassador_config_dir, namespace, envoy_config_file, delay, pid)

    if mode == "sync":
        sync(restarter)
    elif mode == "watch":
        restarter.start()


        while True:
            try:
                # this is in a loop because sometimes the auth expires
                # or the connection dies
                watch_loop(restarter)
            except KeyboardInterrupt:
                raise
            except:
                traceback.print_exc()
    else:
         raise ValueError(mode)

if __name__ == "__main__":
    main()
