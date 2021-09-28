#!/usr/bin/python

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
# This is a debugging and testing tool that simulates the configuration
# cycle of watt -> watch_hook -> Ambassador, given a set of Kubernetes
# inputs. It's the basis of KAT local mode, and also a primary development
# tool at Datawire.
########

from typing import Any, Dict, List, Optional, Tuple, TYPE_CHECKING

import sys

import difflib
import errno
import filecmp
import functools
import io
import json
import logging
import os
import shlex
import shutil

import click
from watch_hook import WatchHook

# Use this instead of click.option
click_option = functools.partial(click.option, show_default=True)
click_option_no_default = functools.partial(click.option, show_default=False)

from ambassador import Config, IR, Diagnostics, EnvoyConfig
from ambassador.fetch import ResourceFetcher
from ambassador.utils import parse_yaml, SecretHandler, SecretInfo, dump_json, parse_bool
from kat.utils import ShellCommand

if TYPE_CHECKING:
    from ambassador.ir import IRResource

KubeResource = Dict[str, Any]
KubeList = List[KubeResource]
WattDict = Dict[str, KubeList]

class LabelSpec:
    def __init__(self, serialization: str) -> None:
        if '=' not in serialization:
            raise Exception(f"label serialization must be key=value, not {serialization}")

        (key, value) = serialization.split('=', 1)

        self.key = key
        self.value = value

    def __str__(self) -> str:
        return f"{self.key}={self.value}"

    def match(self, labels: Dict[str, str]) -> bool:
        return bool(labels.get(self.key, None) == self.value)


class FieldSpec:
    def __init__(self, serialization: str) -> None:
        if '=' not in serialization:
            raise Exception(f"field serialization must be key=value, not {serialization}")

        (key, value) = serialization.split('=', 1)

        self.elements = key.split('.')
        self.value = value

    def __str__(self) -> str:
        return f"{'.'.join(self.elements)}={self.value}"

    def match(self, resource: Dict[str, Any]) -> bool:
        node = resource

        for el in self.elements[:-1]:
            node = node.get(el, None)

            if node is None:
                return False

        return bool(node.get(self.elements[-1], None) == self.value)


class WatchResult:
    def __init__(self, kind: str, watch_id: str) -> None:
        self.kind = kind
        self.watch_id = watch_id

class WatchSpec:
    def __init__(self, logger: logging.Logger, kind: str, namespace: Optional[str],
                 labels: Optional[str], fields: Optional[str]=None,
                 bootstrap: Optional[bool]=False):
        self.logger = logger
        self.kind = kind
        self.match_kinds = { self.kind.lower(): True }
        self.namespace = namespace
        self.labels: Optional[List[LabelSpec]] = None
        self.fields: Optional[List[FieldSpec]] = None
        self.bootstrap = bootstrap

        if self.kind == 'ingresses':
            self.match_kinds['ingress'] = True

        if labels:
            self.labels = [ LabelSpec(l) for l in labels.split(',') ]

        if fields:
            self.fields = [ FieldSpec(f) for f in fields.split(',') ]

    def _labelstr(self) -> str:
        return ",".join([ str(x) for x in self.labels or [] ])

    def _fieldstr(self) -> str:
        return ",".join([ str(x) for x in self.fields or [] ])

    @staticmethod
    def _star(s: Optional[str]) -> str:
        return s if s else "*"

    def __repr__(self) -> str:
        s = f"{self.kind}|{self._star(self.namespace)}|{self._star(self._fieldstr())}|{self._star(self._labelstr())}"

        if self.bootstrap:
            s += ' (bootstrap)'

        return f"<{s}>"

    def __str__(self) -> str:
        if self.bootstrap:
            return f"{self.kind}|bootstrap"
        else:
            return f"{self.kind}|{self._star(self.namespace)}|{self._star(self._fieldstr())}|{self._star(self._labelstr())}"

    def match(self, obj: KubeResource) -> Optional[WatchResult]:
        kind: Optional[str] = obj.get('kind') or None
        metadata: Dict[str, Any] = obj.get('metadata') or {}
        name: Optional[str] = metadata.get('name') or None
        namespace: str = metadata.get('namespace') or 'default'
        labels: Dict[str, str] = metadata.get('labels') or {}

        if not kind or not name:
            self.logger.error(f"K8s object requires kind and name: {obj}")
            return None

        # self.logger.debug(f"match {self}: check {obj}")
        match_kind_str = ','.join(sorted(self.match_kinds.keys()))

        # OK. Does the kind match up?
        if kind.lower() not in self.match_kinds:
            # self.logger.debug(f"match {self}: mismatch for kind {kind}, match_kinds {match_kind_str}")
            return None

        # How about namespace (if present)?
        if self.namespace:
            if namespace != self.namespace:
                # self.logger.debug(f"match {self}: mismatch for namespace {namespace}")
                return None

        # OK, check labels...
        if self.labels:
            for l in self.labels:
                if not l.match(labels):
                    # self.logger.debug(f"match {self}: mismatch for label {l}")
                    return None

        # ...and fields.
        if self.fields:
            for f in self.fields:
                if not f.match(obj):
                    # self.logger.debug(f"match {self}: mismatch for field {f}")
                    return None

        # Woo, it worked!
        self.logger.debug(f"match {self} - {match_kind_str}: good!")
        # self.logger.debug(f"{obj}")

        return WatchResult(kind=self.kind, watch_id=str(self))


class Mockery:
    def __init__(self, logger: logging.Logger, debug: bool, sources: List[str],
                 labels: Optional[str], namespace: Optional[str], watch: str) -> None:
        self.logger = logger
        self.debug = debug
        self.sources = sources
        self.namespace = namespace
        self.watch = watch

        self.watch_specs: Dict[str, WatchSpec] = {}

        # Set up bootstrap sources.
        for source in sources:
            bootstrap_watch = WatchSpec(
                logger=self.logger,
                kind=source,
                namespace=self.namespace,
                labels=labels,
                bootstrap=True
            )

            if not self.maybe_add(bootstrap_watch):
                self.logger.error(f"how is a bootstrap watch not new? {bootstrap_watch}")
                sys.exit(1)

    def maybe_add(self, w: WatchSpec) -> bool:
        key = str(w)

        if key in self.watch_specs:
            return False
        else:
            self.watch_specs[key] = w

        return True

    def load(self, manifest: KubeList) -> WattDict:
        collected: Dict[str, Dict[str, KubeResource]] = {}
        watt_k8s: WattDict = {}

        self.logger.info("LOADING:")

        for spec in self.watch_specs.values():
            self.logger.debug(f"{repr(spec)}")

        for obj in manifest:
            metadata = obj.get('metadata') or {}
            name = metadata.get('name')

            if not name:
                self.logger.debug(f"skipping unnamed object {obj}")
                continue

            # self.logger.debug(f"consider {obj}")

            for w in self.watch_specs.values():
                m = w.match(obj)

                if m:
                    by_type = collected.setdefault(m.kind, {})

                    # If we already have this object's name in the collection,
                    # this is a duplicate find.
                    if name not in by_type:
                        by_type[name] = obj

        # Once that's all done, flatten everything.
        for kind in collected.keys():
            watt_k8s[kind] = list(collected[kind].values())

        self.snapshot = dump_json({ 'Consul': {}, 'Kubernetes': watt_k8s }, pretty=True)

        return watt_k8s

    def run_hook(self) -> Tuple[bool, bool]:
        self.logger.info("RUNNING HOOK")

        yaml_stream = io.StringIO(self.snapshot)

        wh = WatchHook(self.logger, yaml_stream)

        any_changes = False

        if wh.watchset:
            for w in wh.watchset.get("kubernetes-watches") or []:
                potential = WatchSpec(
                    logger=self.logger,
                    kind=w['kind'],
                    namespace=w.get('namespace'),
                    labels=w.get('label-selector'),
                    fields=w.get('field-selector'),
                    bootstrap=False
                )

                if self.maybe_add(potential):
                    any_changes = True

        return True, any_changes


class MockSecretHandler(SecretHandler):
    def load_secret(self, resource: 'IRResource', secret_name: str, namespace: str) -> Optional[SecretInfo]:
        # Allow an environment variable to state whether we're in Edge Stack. But keep the
        # existing condition as sufficient, so that there is less of a chance of breaking
        # things running in a container with this file present.
        if parse_bool(os.environ.get('EDGE_STACK', 'false')) or os.path.exists('/ambassador/.edge_stack'):
            if ((secret_name == "fallback-self-signed-cert") and
                (namespace == Config.ambassador_namespace)):
                # This is Edge Stack. Force the fake TLS secret.

                self.logger.info(f"MockSecretHandler: mocking fallback secret {secret_name}.{namespace}")
                return SecretInfo(secret_name, namespace, "mocked-fallback-secret",
                                  "-fallback-cert-", "-fallback-key-", decode_b64=False)

        self.logger.debug(f"MockSecretHandler: cannot load {secret_name}.{namespace}")
        return None

@click.command(help="Mock the watt/watch_hook/diagd cycle to generate an IR from a Kubernetes YAML manifest.")
@click_option('--debug/--no-debug', default=True,
              help="enable debugging")
@click_option('-n', '--namespace', type=click.STRING,
              help="namespace to watch [default: all namespaces])")
@click_option('-s', '--source', type=click.STRING, multiple=True,
              help="define initial source types [default: all Ambassador resources]")
@click_option('--labels', type=click.STRING, multiple=True,
              help="define initial label selector")
@click_option('--force-pod-labels/--no-force-pod-labels', default=True,
              help="copy initial label selector to /tmp/ambassador-pod-info/labels")
@click_option('--kat-name', '--kat', type=click.STRING,
              help="emulate a running KAT test with this name")
@click_option('-w', '--watch', type=click.STRING, default="python /ambassador/watch_hook.py",
              help="define a watch hook")
@click_option('--diff-path', '--diff', type=click.STRING,
              help="directory to diff against")
@click_option('--include-ir/--no-include-ir', '--ir/--no-ir', default=False,
              help="include IR in diff when using --diff-path")
@click_option('--include-aconf/--no-include-aconf', '--aconf/--no-aconf', default=False,
              help="include AConf in diff when using --diff-path")
@click_option('--update/--no-update', default=False,
              help="update the diff path when finished")
@click.argument('k8s-yaml-paths', nargs=-1)
def main(k8s_yaml_paths: List[str], debug: bool, force_pod_labels: bool, update: bool,
         source: List[str], labels: List[str], namespace: Optional[str], watch: str,
         include_ir: bool, include_aconf: bool,
         diff_path: Optional[str]=None, kat_name: Optional[str]=None) -> None:
    loglevel = logging.DEBUG if debug else logging.INFO

    logging.basicConfig(
        level=loglevel,
        format="%(asctime)s mockery %(levelname)s: %(message)s",
        datefmt="%Y-%m-%d %H:%M:%S"
    )

    logger = logging.getLogger('mockery')

    logger.debug(f"reading from {k8s_yaml_paths}")

    if not source:
        source = [
            "AmbassadorHost", "service", "ingresses",
            "AuthService", "AmbassadorListener", "LogService", "AmbassadorMapping", "Module", "RateLimitService",
            "TCPMapping", "TLSContext", "TracingService",
            "ConsulResolver", "KubernetesEndpointResolver", "KubernetesServiceResolver"
        ]

    if namespace:
        os.environ['AMBASSADOR_NAMESPACE'] = namespace

    # Make labels a list, instead of a tuple.
    labels = list(labels)
    labels_to_force = { l: True for l in labels or [] }

    if kat_name:
        logger.debug(f"KAT name {kat_name}")

        # First set up some labels to force.

        labels_to_force["scope=AmbassadorTest"] = True
        labels_to_force[f"service={kat_name}"] = True

        kat_amb_id_label = f"kat-ambassador-id={kat_name}"

        if kat_amb_id_label not in labels_to_force:
            labels_to_force[kat_amb_id_label] = True
            labels.append(kat_amb_id_label)

        os.environ['AMBASSADOR_ID'] = kat_name
        os.environ['AMBASSADOR_LABEL_SELECTOR'] = kat_amb_id_label

        # Forcibly override the cached ambassador_id.
        Config.ambassador_id = kat_name

    logger.debug(f"namespace {namespace or '*'}")
    logger.debug(f"labels to watch {', '.join(labels)}")
    logger.debug(f"labels to force {', '.join(sorted(labels_to_force.keys()))}")
    logger.debug(f"watch hook {watch}")
    logger.debug(f"sources {', '.join(source)}")

    for key in sorted(os.environ.keys()):
        if key.startswith('AMBASSADOR'):
            logger.debug(f"${key}={os.environ[key]}")

    if force_pod_labels:
        try:
            os.makedirs("/tmp/ambassador-pod-info")
        except OSError as e:
            if e.errno != errno.EEXIST:
                raise

        with open("/tmp/ambassador-pod-info/labels", "w", encoding="utf-8") as outfile:
            for l in labels_to_force:
                outfile.write(l)
                outfile.write("\n")

    # Pull in the YAML.
    input_yaml = ''.join([ open(x, "r").read() for x in k8s_yaml_paths ])
    manifest = parse_yaml(input_yaml)

    w = Mockery(logger, debug, source, ",".join(labels), namespace, watch)

    iteration = 0

    while True:
        iteration += 1

        if iteration > 10:
            print(f"!!!! Not stable after 10 iterations, failing")
            logger.error("Not stable after 10 iterations, failing")
            sys.exit(1)

        logger.info(f"======== START ITERATION {iteration}")

        w.load(manifest)

        logger.info(f"WATT_K8S: {w.snapshot}")

        hook_ok, any_changes = w.run_hook()

        if not hook_ok:
            raise Exception("hook failed")

        if any_changes:
            logger.info(f"======== END ITERATION {iteration}: watches changed!")
        else:
            logger.info(f"======== END ITERATION {iteration}: stable!")
            break

    # Once here, we should be good to go.
    try:
        os.makedirs("/tmp/ambassador/snapshots")
    except OSError as e:
        if e.errno != errno.EEXIST:
            raise

    scc = MockSecretHandler(logger, "mockery", "/tmp/ambassador/snapshots", f"v{iteration}")

    aconf = Config()

    logger.debug(f"Config.ambassador_id {Config.ambassador_id}")
    logger.debug(f"Config.ambassador_namespace {Config.ambassador_namespace}")

    logger.info(f"STABLE WATT_K8S: {w.snapshot}")

    fetcher = ResourceFetcher(logger, aconf)
    fetcher.parse_watt(w.snapshot)
    aconf.load_all(fetcher.sorted())

    open("/tmp/ambassador/snapshots/aconf.json", "w", encoding="utf-8").write(aconf.as_json())

    ir = IR(aconf, secret_handler=scc)

    open("/tmp/ambassador/snapshots/ir.json", "w", encoding="utf-8").write(ir.as_json())

    econf = EnvoyConfig.generate(ir, Config.envoy_api_version)
    bootstrap_config, ads_config, clustermap = econf.split_config()

    ads_config.pop('@type', None)
    with open("/tmp/ambassador/snapshots/econf.json", "w", encoding="utf-8") as outfile:
        outfile.write(dump_json(ads_config, pretty=True))

    with open(f"/tmp/ambassador/snapshots/econf-{Config.ambassador_id}.json", "w", encoding="utf-8") as outfile:
        outfile.write(dump_json(ads_config, pretty=True))

    with open("/tmp/ambassador/snapshots/bootstrap.json", "w", encoding="utf-8") as outfile:
        outfile.write(dump_json(bootstrap_config, pretty=True))

    diag = Diagnostics(ir, econf)

    with open("/tmp/ambassador/snapshots/diag.json", "w", encoding="utf-8") as outfile:
        outfile.write(dump_json(diag.as_dict(), pretty=True))

    if diff_path:
        diffs = False

        pairs_to_check = [
            (os.path.join(diff_path, 'snapshots', 'econf.json'), '/tmp/ambassador/snapshots/econf.json'),
            (os.path.join(diff_path, 'bootstrap-ads.json'), '/tmp/ambassador/snapshots/bootstrap.json')
        ]

        if include_ir:
            pairs_to_check.append(
                ( os.path.join(diff_path, 'snapshots', 'ir.json'), '/tmp/ambassador/snapshots/ir.json' )
            )

        if include_aconf:
            pairs_to_check.append(
                ( os.path.join(diff_path, 'snapshots', 'aconf.json'), '/tmp/ambassador/snapshots/aconf.json' )
            )

        for gold_path, check_path in pairs_to_check:
            if update:
                logger.info(f"mv {check_path} {gold_path}")
                shutil.move(check_path, gold_path)
            elif not filecmp.cmp(gold_path, check_path):
                diffs = True

                gold_lines = open(gold_path, "r", encoding="utf-8").readlines()
                check_lines = open(check_path, "r", encoding="utf-8").readlines()

                for line in difflib.unified_diff(gold_lines, check_lines, fromfile=gold_path, tofile=check_path):
                    sys.stdout.write(line)

        if diffs:
            sys.exit(1)


if __name__ == '__main__':
    main()
