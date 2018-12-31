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

from typing import Dict, Optional, Union
from typing import cast as typecast

import sys

import binascii
import json
import logging
import os
import signal
import traceback
import yaml

import clize
from clize import Parameter

from . import Scout, Config, IR, Diagnostics, Version
from .envoy import EnvoyConfig, V1Config, V2Config
from .ir.irtlscontext import IRTLSContext

from .utils import RichStatus, SavedSecret

__version__ = Version

logging.basicConfig(
    level=logging.INFO,
    format="%%(asctime)s ambassador-cli %s %%(levelname)s: %%(message)s" % __version__,
    datefmt="%Y-%m-%d %H:%M:%S"
)

logger = logging.getLogger("ambassador")


def handle_exception(what, e, **kwargs):
    tb = "\n".join(traceback.format_exception(*sys.exc_info()))

    scout = Scout()
    result = scout.report(action=what, mode="cli", exception=str(e), traceback=tb, **kwargs)

    logger.debug("Scout %s, result: %s" %
                 ("enabled" if scout._scout else "disabled", result))

    logger.error("%s: %s\n%s" % (what, e, tb))

    show_notices(result)


def show_notices(result: dict, printer=logger.log):
    notices = result.get('notices', [])

    for notice in notices:
        lvl = logging.getLevelName(notice.get('level', 'ERROR'))

        printer(lvl, notice.get('message', '?????'))


def stdout_printer(lvl, msg):
    print("%s: %s" % (logging.getLevelName(lvl), msg))


def version():
    """
    Show Ambassador's version
    """

    print("Ambassador %s" % __version__)

    scout = Scout()

    print("Ambassador Scout version %s" % scout.version)
    print("Ambassador Scout semver  %s" % scout.get_semver(scout.version))

    result = scout.report(action="version", mode="cli")
    show_notices(result, printer=stdout_printer)


def showid():
    """
    Show Ambassador's installation ID
    """

    scout = Scout()

    print("Ambassador Scout installation ID %s" % scout.install_id)

    result= scout.report(action="showid", mode="cli")
    show_notices(result, printer=stdout_printer)


def file_checker(path: str) -> bool:
    logger.debug("CLI file checker: pretending %s exists" % path)
    return True


def cli_secret_reader(context: IRTLSContext, secret_name: str, namespace: str, secret_root: str) -> SavedSecret:
    # In the Real World, the secret reader should, y'know, read secrets..
    # Here we're just gonna fake it.

    cert_path = os.path.join(secret_root, namespace, "cli-secrets", secret_name, "tls.crt")
    key_path = os.path.join(secret_root, namespace, "cli-secrets", secret_name, "tls.key")

    return SavedSecret(secret_name, namespace, cert_path, key_path, {})


def dump(config_dir_path: Parameter.REQUIRED, *,
         debug=False, debug_scout=False, k8s=False, recurse=False,
         aconf=False, ir=False, v1=False, v2=False, diag=False, features=False):
    """
    Dump various forms of an Ambassador configuration for debugging

    Use --aconf, --ir, and --envoy to control what gets dumped. If none are requested, the IR
    will be dumped.

    :param config_dir_path: Configuration directory to scan for Ambassador YAML files
    :param debug: If set, generate debugging output
    :param debug_scout: If set, generate debugging output
    :param k8s: If set, assume configuration files are annotated K8s manifests
    :param recurse: If set, recurse into directories below config_dir_path
    :param aconf: If set, dump the Ambassador config
    :param ir: If set, dump the IR
    :param v1: If set, dump the Envoy V1 config
    :param v2: If set, dump the Envoy V2 config
    :param diag: If set, dump the Diagnostics overview
    :param features: If set, dump the feature set
    """

    if debug:
        logger.setLevel(logging.DEBUG)

    if debug_scout:
        logging.getLogger('ambassador.scout').setLevel(logging.DEBUG)

    if not (aconf or ir or v1 or v2 or diag or features):
        aconf = True
        ir = True
        v1 = False  # Default to NOT dumping V1 any more.
        v2 = True
        diag = False
        features = False

    dump_aconf = aconf
    dump_ir = ir
    dump_v1 = v1
    dump_v2 = v2
    dump_diag = diag
    dump_features = features

    od = {}
    diagconfig: Optional[EnvoyConfig] = None

    try:
        aconf = Config()
        aconf.load_from_directory(config_dir_path, k8s=k8s, recurse=recurse)
        # aconf.post_error("Error from string, boo yah")
        # aconf.post_error(RichStatus.fromError("Error from RichStatus"))

        if dump_aconf:
            od['aconf'] = aconf.as_dict()

        ir = IR(aconf, file_checker=file_checker, secret_reader=cli_secret_reader)

        if dump_ir:
            od['ir'] = ir.as_dict()

        if dump_v1:
            v1config = V1Config(ir)
            diagconfig = v1config
            od['v1'] = v1config.as_dict()

        if dump_v2:
            v2config = V2Config(ir)
            diagconfig = v2config
            od['v2'] = v2config.as_dict()

        if dump_diag:
            econf = typecast(EnvoyConfig, diagconfig)
            diag = Diagnostics(ir, econf)
            od['diag'] = diag.as_dict()
            od['elements'] = econf.elements

        if dump_features:
            od['features'] = ir.features()

        scout = Scout()
        scout_args = {}

        if ir and not os.environ.get("AMBASSADOR_DISABLE_FEATURES", None):
            scout_args["features"] = ir.features()

        result = scout.report(action="dump", mode="cli", **scout_args)
        show_notices(result)

        json.dump(od, sys.stdout, sort_keys=True, indent=4)
        sys.stdout.write("\n")
    except Exception as e:
        handle_exception("EXCEPTION from dump", e,
                         config_dir_path=config_dir_path)

        # This is fatal.
        sys.exit(1)


def validate(config_dir_path: Parameter.REQUIRED, **kwargs):
    """
    Validate an Ambassador configuration. This is an extension of "config" that
    redirects output to devnull and always exits on error.

    :param config_dir_path: Configuration directory to scan for Ambassador YAML files
    """
    config(config_dir_path, os.devnull, exit_on_error=True, **kwargs)


def config(config_dir_path: Parameter.REQUIRED, output_json_path: Parameter.REQUIRED, *,
           debug=False, debug_scout=False, check=False, k8s=False, ir=None, aconf=None,
           exit_on_error=False, v1=False, v2=False):
    """
    Generate an Envoy configuration

    :param config_dir_path: Configuration directory to scan for Ambassador YAML files
    :param output_json_path: Path to output envoy.json
    :param debug: If set, generate debugging output
    :param debug_scout: If set, generate debugging output when talking to Scout
    :param check: If set, generate configuration only if it doesn't already exist
    :param k8s: If set, assume configuration files are annotated K8s manifests
    :param exit_on_error: If set, will exit with status 1 on any configuration error
    :param ir: Pathname to which to dump the IR (not dumped if not present)
    :param aconf: Pathname to which to dump the aconf (not dumped if not present)
    :param v1: If set, output config to v1 envoy config
    :param v2: If set, output config to v2 envoy config (default)
    """
    if debug:
        logger.setLevel(logging.DEBUG)

    if debug_scout:
        logging.getLogger('ambassador.scout').setLevel(logging.DEBUG)

    try:
        logger.debug("CHECK MODE  %s" % check)
        logger.debug("CONFIG DIR  %s" % config_dir_path)
        logger.debug("OUTPUT PATH %s" % output_json_path)

        dump_aconf: Optional[str] = aconf
        dump_ir: Optional[str] = ir

        # Bypass the existence check...
        output_exists = False

        if check:
            # ...oh no wait, they explicitly asked for the existence check!
            # Assume that the file exists (ie, we'll do nothing) unless we
            # determine otherwise.
            output_exists = True

            try:
                json.loads(open(output_json_path, "r").read())
            except FileNotFoundError:
                logger.debug("output file does not exist")
                output_exists = False
            except OSError:
                logger.warning("output file is not sane?")
                output_exists = False
            except json.decoder.JSONDecodeError:
                logger.warning("output file is not valid JSON")
                output_exists = False

            logger.info("Output file %s" % ("exists" if output_exists else "does not exist"))

        rc = RichStatus.fromError("impossible error")

        if not output_exists:
            # Either we didn't need to check, or the check didn't turn up
            # a valid config. Regenerate.
            logger.info("Generating new Envoy configuration...")

            aconf = Config()
            aconf.load_from_directory(config_dir_path, k8s=k8s)

            if dump_aconf:
                with open(dump_aconf, "w") as output:
                    output.write(aconf.as_json())
                    output.write("\n")

            # If exit_on_error is set, log _errors and exit with status 1
            if exit_on_error and aconf.errors:
                raise Exception("errors in: {0}".format(', '.join(aconf.errors.keys())))

            ir = IR(aconf, file_checker=file_checker, secret_reader=cli_secret_reader)

            if dump_ir:
                with open(dump_ir, "w") as output:
                    output.write(ir.as_json())
                    output.write("\n")

            # clize considers kwargs with False for default value as flags,
            # resulting in the logic below.
            # https://clize.readthedocs.io/en/stable/basics.html#accepting-flags

            # Flag --v1 is specified, it takes precedence over default --v2.
            # Make sure they are mutually exclusive, defaulting to v2 if both are True.
            if v1 and not v2:
                logger.info("Writing envoy V1 configuration")
                v1config = V1Config(ir)
                rc = RichStatus.OK(msg="huh")

                if rc:
                    with open(output_json_path, "w") as output:
                        output.write(v1config.as_json())
                        output.write("\n")
                else:
                    logger.error("Could not generate new Envoy configuration: %s" % rc.error)
            # Flag --v1 is not specified generate v2 config.
            else:
                logger.info("Writing envoy V2 configuration")
                v2config = V2Config(ir)
                rc = RichStatus.OK(msg="huh_v2")

                if rc:
                    with open(output_json_path, "w") as output:
                        output.write(v2config.as_json())
                        output.write("\n")
                else:
                    logger.error("Could not generate new Envoy configuration: %s" % rc.error)

        scout = Scout()
        result = scout.report(action="config", mode="cli")
        show_notices(result)
    except Exception as e:
        handle_exception("EXCEPTION from config", e,
                         config_dir_path=config_dir_path, output_json_path=output_json_path)

        # This is fatal.
        sys.exit(1)


class SplitConfigChecker:
    def __init__(self, logger, root_path: str) -> None:
        self.logger = logger
        self.root = root_path

    def secret_reader(self, context: 'IRTLSContext', secret_name: str, namespace: str, secret_root: str):
        yaml_path = os.path.join(self.root, namespace, "secrets", "%s.yaml" % secret_name)

        serialization = None
        objects = []
        cert_data = None
        cert = None
        key = None
        cert_path = None
        key_path = None

        try:
            serialization = open(yaml_path, "r").read()
        except IOError as e:
            self.logger.error("TLSContext %s: SCC.secret_reader could not open %s" % (context.name, yaml_path))

        if serialization:
            try:
                objects.extend(list(yaml.safe_load_all(serialization)))
            except yaml.error.YAMLError as e:
                self.logger.error("TLSContext %s: SCC.secret_reader could not parse %s: %s" %
                                  (context.name, yaml_path, e))

        ocount = 0
        errors = 0

        for obj in objects:
            ocount += 1
            kind = obj.get('kind', None)

            if kind != "Secret":
                self.logger.error("TLSContext %s: SCC.secret_reader found K8s %s at %s.%d?" %
                                  (context.name, kind, yaml_path, ocount))
                errors += 1
                continue

            metadata = obj.get('metadata', None)

            if not metadata:
                self.logger.error("TLSContext %s: SCC.secret_reader found K8s Secret with no metadata at %s.%d?" %
                                  (context.name, yaml_path, ocount))
                errors += 1
                continue

            if 'data' in obj:
                if cert_data:
                    self.logger.error("TLSContext %s: SCC.secret_reader found multiple Secrets in %s?" %
                                      (context.name, yaml_path))
                    errors += 1
                    continue

                cert_data = obj['data']

        # if errors:
        #     return None
        #
        # if not cert_data:
        #     self.logger.error("TLSContext %s: SCC.secret_reader found no certificate in %s?" %
        #                       (context.name, yaml_path))
        #     return None

        # OK, we have something to work with. Hopefully.
        if not errors and cert_data:
            cert = cert_data.get('tls.crt', None)

            if cert:
                cert = binascii.a2b_base64(cert)

            key = cert_data.get('tls.key', None)

            if key:
                key = binascii.a2b_base64(key)

        # if not cert:
        #     # This is an error. Having a cert but no key might be OK, we'll let our caller decide.
        #     self.logger.error("TLSContext %s: SCC.secret_reader found data but no cert in %s?" %
        #                       (context.name, yaml_path))
        #     return None

        if cert:
            secret_dir = os.path.join(self.root, namespace, "secrets-decoded", secret_name)

            try:
                os.makedirs(secret_dir)
            except FileExistsError:
                pass

            cert_path = os.path.join(secret_dir, "tls.crt")
            open(cert_path, "w").write(cert.decode("utf-8"))

            if key:
                key_path = os.path.join(secret_dir, "tls.key")
                open(key_path, "w").write(key.decode("utf-8"))

        return SavedSecret(secret_name, namespace, cert_path, key_path, cert_data)


def splitconfig(root_path: Parameter.REQUIRED, *, ambex_pid: int=0,
                bootstrap_path=None, ads_path=None, changes=None, gencount=None,
                debug=False, debug_scout=False, k8s=True, ir_path=None):
    """
    Generate an Envoy configuration from resources that have already been pulled from Kube

    :param root_path: Root of the config data. Expected to contain subdirs for namespaces
    :param ambex_pid: PID of running Ambex to signal on config changes
    :param bootstrap_path: Path to which to write Envoy bootstrap config
    :param ads_path: Path to which to write Envoy ADS config for Ambex
    :param ir_path: If set, path to which to dump the IR
    :param changes: How many changes since the last update have happened?
    :param gencount: Generation count of this update
    :param debug: If set, generate debugging output
    :param debug_scout: If set, generate debugging output when talking to Scout
    """
    # :param k8s: If set, assume configuration files are annotated K8s manifests
    # """

    if debug:
        logger.setLevel(logging.DEBUG)

    if debug_scout:
        logging.getLogger('ambassador.scout').setLevel(logging.DEBUG)

    # root_path contains directories for each resource type: services, secrets, optional
    # crd-whatever paths.
    scc = SplitConfigChecker(logger, root_path)

    # Start by assuming that we're going to look at everything.
    config_root = root_path

    # ...then override that if we're running in single-namespace mode.
    if os.environ.get('AMBASSADOR_SINGLE_NAMESPACE'):
        config_root = os.path.join(root_path, os.environ.get('AMBASSADOR_NAMESPACE', 'default'))

    # OK. Load the Config from the config_root.
    aconf = Config()
    aconf.load_from_directory(config_root, k8s=k8s, recurse=True)

    # Use the SplitConfigChecker to resolve secrets. We don't pass a file checker
    # because anything in the config using an actual path needs to be passing a
    # correct path by this point.
    ir = IR(aconf, secret_reader=scc.secret_reader)

    # Generate a V2Config from that, and grab the split bootstrap and ADS configs.
    v2config = V2Config(ir)
    bootstrap_config, ads_config = v2config.split_config()

    if not bootstrap_path:
        bootstrap_path="bootstrap-ads.json"

    if not ads_path:
        ads_path="envoy/envoy.json"

    logger.info("SAVING CONFIG")

    with open(bootstrap_path, "w") as output:
        output.write(json.dumps(bootstrap_config, sort_keys=True, indent=4))

    with open(ads_path, "w") as output:
        output.write(json.dumps(ads_config, sort_keys=True, indent=4))

    if ir_path:
        with open(ir_path, "w") as output:
            output.write(ir.as_json())

    if ambex_pid != 0:
        logger.info("RESTARTING")
        os.kill(ambex_pid, signal.SIGHUP)


def main():
    clize.run([config, splitconfig, dump, validate], alt=[version, showid],
              description="""
              Generate an Envoy config, or manage an Ambassador deployment. Use

              ambassador.py command --help

              for more help, or

              ambassador.py --version

              to see Ambassador's version.
              """)


if __name__ == "__main__":
    main()
