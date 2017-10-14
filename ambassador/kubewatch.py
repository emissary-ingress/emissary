import click, os, shutil, signal, subprocess, sys, threading, time, traceback
from kubernetes import client, config, watch
from AmbassadorConfig import AmbassadorConfig

# XXX: need to figure out the proper conventions for annotation keys
KEY = "ambassador"

def is_annotated(svc):
    annotations = svc.metadata.annotations
    return annotations and KEY in annotations

def get_annotation(svc):
    return svc.metadata.annotations[KEY] if is_annotated(svc) else None

def get_filename(svc):
    return "%s.yaml" % svc.metadata.name

class Restarter(threading.Thread):

    def __init__(self, ambassador_config_dir, envoy_config_file, delay, pid):
        threading.Thread.__init__(self, daemon=True)

        self.ambassador_config_dir = ambassador_config_dir
        self.envoy_config_file = envoy_config_file
        self.delay = delay
        self.pid = pid

        self.mutex = threading.Condition()
        # This holds how many times we have been poked.
        self.pokes = 0
        # This holds how many pokes we have actually processed.
        self.processed = self.pokes
        self.restart_count = 0

        while True:
            if not os.path.exists("%s-%s" % (self.ambassador_config_dir, self.restart_count + 1)):
                break
            else:
                self.restart_count += 1

        self.configs = {}
        path = "%s-%s" % (self.ambassador_config_dir, self.restart_count)
        if os.path.exists(path):
            print ("Restoring config inputs from %s" % path)
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

        aconf = AmbassadorConfig(output)
        rc = aconf.generate_envoy_config()

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

    def update(self, svc):
        config = get_annotation(svc)
        if config is None:
            self.delete(svc)
        else:
            key = get_filename(svc)
            with self.mutex:
                if key in self.configs:
                    if config != self.configs[key]:
                        self.configs[key] = config
                        self.poke()
                elif key not in self.configs:
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
    # XXX: is there a better way to check if we are inside a cluster or not?
    if "KUBERNETES_SERVICE_HOST" in os.environ:
        config.load_incluster_config()
    else:
        config.load_kube_config()

    return client.CoreV1Api()

def sync(restarter):
    v1 = kube_v1()
    for svc in v1.list_service_for_all_namespaces().items:
        restarter.update(svc)
    if restarter.changes():
        print ("Changes detected, regenerating envoy config.")
        restarter.restart()
    else:
        print ("No changes detected, no regen needed.")

def watch_loop(restarter):
    v1 = kube_v1()
    w = watch.Watch()
    for evt in w.stream(v1.list_service_for_all_namespaces):
        print("Event: %s %s" % (evt["type"], evt["object"].metadata.name))
        if evt["type"] == "DELETED":
            restarter.delete(evt["object"])
        else:
            restarter.update(evt["object"])

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
    restarter = Restarter(ambassador_config_dir, envoy_config_file, delay, pid)

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
