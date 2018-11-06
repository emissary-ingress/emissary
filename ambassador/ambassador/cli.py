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

from typing import List, Optional, Dict

import sys

import json
import logging
import os
import traceback

import clize
from clize import Parameter

from . import Scout, ScoutNotice, Config, IR, Diagnostics, Version
from .config import fetch_resources
from .envoy import V1Config
from .envoy import V2Config

from .utils import RichStatus

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


def tls_secret_resolver(secret_name: str, context: str, cert_dir=None) -> Optional[Dict[str, str]]:
    # In the Real World, kubewatch hands in a resolver that looks into kubernetes.
    # Here we're just gonna fake it.

    if context == 'server':
        return {
            'cert_chain_file': "/path/to/%s.crt" % secret_name,
            'private_key_file': "/path/to/%s.key" % secret_name
        }
    elif context == 'client':
        return {
            'cacert_chain_file': "/path/to/%s.crt" % secret_name,
        }
    else:
        if cert_dir is not None:
            return {
                'certificate_chain_file': os.path.join(cert_dir, 'cert.crt'),
                'private_key_file': os.path.join(cert_dir, 'cert.key')
            }
        return None


def dump(config_dir_path: Parameter.REQUIRED, *,
         debug=False, debug_scout=False, k8s=False,
         aconf=False, ir=False, v1=False, v2=False, diag=False):
    """
    Dump various forms of an Ambassador configuration for debugging

    Use --aconf, --ir, and --envoy to control what gets dumped. If none are requested, the IR
    will be dumped.

    :param config_dir_path: Configuration directory to scan for Ambassador YAML files
    :param k8s: If set, assume configuration files are annotated K8s manifests
    :param debug: If set, generate debugging output
    :param debug_scout: If set, generate debugging output
    :param aconf: If set, dump the Ambassador config
    :param ir: If set, dump the IR
    :param v1: If set, dump the Envoy V1 config
    :param v2: If set, dump the Envoy V2 config
    :param diag: If set, dump the Diagnostics overview
    """

    if debug:
        logger.setLevel(logging.DEBUG)

    if debug_scout:
        logging.getLogger('ambassador.scout').setLevel(logging.DEBUG)

    if not (aconf or ir or v1 or v2 or diag):
        aconf = True
        ir = True
        v1 = False  # Default to NOT dumping V1 any more.
        v2 = True
        diag = False

    dump_aconf = aconf
    dump_ir = ir
    dump_v1 = v1
    dump_v2 = v2
    dump_diag = diag

    od = {}
    diagconfig = None

    try:
        resources = fetch_resources(config_dir_path, logger, k8s=k8s)
        aconf = Config()
        aconf.load_all(resources)

        if dump_aconf:
            od['aconf'] = aconf.as_dict()

        ir = IR(aconf, tls_secret_resolver=tls_secret_resolver)

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
            diag = Diagnostics(ir, diagconfig)
            od['diag'] = diag.as_dict()
            od['elements'] = diagconfig.elements

        scout = Scout()
        result = scout.report(action="dump", mode="cli")
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
    :param debug_scout: If set, generate debugging output
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

            resources = fetch_resources(config_dir_path, logger, k8s=k8s)
            aconf = Config()
            aconf.load_all(resources)

            if dump_aconf:
                with open(dump_aconf, "w") as output:
                    output.write(aconf.as_json())
                    output.write("\n")

            # If exit_on_error is set, log _errors and exit with status 1
            if exit_on_error and aconf.errors:
                raise Exception("errors in: {0}".format(', '.join(aconf.errors.keys())))

            ir = IR(aconf, tls_secret_resolver=tls_secret_resolver)

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


def main():
    clize.run([config, dump, validate], alt=[version, showid],
              description="""
              Generate an Envoy config, or manage an Ambassador deployment. Use

              ambassador.py command --help

              for more help, or

              ambassador.py --version

              to see Ambassador's version.
              """)


if __name__ == "__main__":
    main()
