import sys

import json
import logging
import os
import semantic_version
import time
import traceback
import uuid

import clize
from clize import Parameter
from scout import Scout

from AmbassadorConfig import AmbassadorConfig

import VERSION

__version__ = VERSION.Version

logging.basicConfig(
    level=logging.INFO, # if appDebug else logging.INFO,
    format="%%(asctime)s ambassador %s %%(levelname)s: %%(message)s" % __version__,
    datefmt="%Y-%m-%d %H:%M:%S"
)

# logging.getLogger("datawire.scout").setLevel(logging.DEBUG)
logger = logging.getLogger("ambassador")
logger.setLevel(logging.DEBUG)

# Weird stuff. The build version looks like
#
# 0.12.0                    for a prod build, or
# 0.12.1-b2.da5d895.DIRTY   for a dev build (in this case made from a dirty true)
#
# Now: 
# - Scout needs a build number (semver "+something") to flag a non-prod release;
#   but
# - DockerHub cannot use a build number at all; but
# - 0.12.1-b2 comes _before_ 0.12.1+b2 in SemVer land.
# 
# FFS.
#
# We cope with this by transforming e.g.
#
# 0.12.1-b2.da5d895.DIRTY into 0.12.1-b2+da5d895.DIRTY
#
# for Scout.

scout_version = __version__

if '-' in scout_version:
    # Dev build!
    v, p = scout_version.split('-')
    p, b = p.split('.', 1) if ('.' in p) else (0, p)

    scout_version = "%s-%s+%s" % (v, p, b)

logger.debug("Scout version %s" % scout_version)

scout = None

runtime = "kubernetes" if os.environ.get('KUBERNETES_SERVICE_HOST', None) else "docker"
logger.debug("runtime: %s" % runtime)

try:
    namespace = os.environ.get('AMBASSADOR_NAMESPACE', 'default')

    scout = Scout(app="ambassador", version=scout_version, 
                  id_plugin=Scout.configmap_install_id_plugin, 
                  id_plugin_args={ "namespace": namespace })
except OSError as e:
    logger.warning("couldn't do version check: %s" % str(e))

def handle_exception(what, e, **kwargs):
    tb = "\n".join(traceback.format_exception(*sys.exc_info()))

    if scout:
        result = scout.report(action=what, exception=str(e), traceback=tb,
                              runtime=runtime, **kwargs)
        logger.debug("Scout %s, result: %s" % ("disabled" if scout.disabled else "enabled", result))

    logger.error("%s: %s\n%s" % (what, e, tb))

def version():
    """
    Show Ambassador's version
    """

    print("Ambassador %s" % __version__)

def showid():
    """
    Show Ambassador's installation ID
    """

    if scout:
        print("%s" % scout.install_id)
    else:
        print("unknown")

def dump(config_dir_path:Parameter.REQUIRED):
    """
    Dump the intermediate form of an Ambassador configuration for debugging

    :param config_dir_path: Configuration directory to scan for Ambassador YAML files
    """
    try:
        logger.debug("CONFIG DIR  %s" % config_dir_path)
        aconf = AmbassadorConfig(config_dir_path)
        json.dump(aconf.envoy_config, sys.stdout, indent=4, sort_keys=True)
    except Exception as e:
        handle_exception("EXCEPTION from dump", e, 
                         config_dir_path=config_dir_path)

def config(config_dir_path:Parameter.REQUIRED, output_json_path:Parameter.REQUIRED, *, check=False):
    """
    Generate an Envoy configuration

    :param config_dir_path: Configuration directory to scan for Ambassador YAML files
    :param output_json_path: Path to output envoy.json
    :param check: If set, generate configuration only if it doesn't already exist
    """

    try:
        logger.debug("CHECK MODE  %s" % check)
        logger.debug("CONFIG DIR  %s" % config_dir_path)
        logger.debug("OUTPUT PATH %s" % output_json_path)

        # Bypass the existence check...
        output_exists = False

        if check:
            # ...oh no wait, they explicitly asked for the existence check!
            # Assume that the file exists (ie, we'll do nothing) unless we
            # determine otherwise.
            output_exists = True

            try:
                x = json.loads(open(output_json_path, "r").read())
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

        if not output_exists:
            # Either we didn't need to check, or the check didn't turn up
            # a valid config. Regenerate.
            logger.info("Generating new Envoy configuration...")
            aconf = AmbassadorConfig(config_dir_path)
            rc = aconf.generate_envoy_config()

            if rc:
                aconf.pretty(rc.envoy_config, out=open(output_json_path, "w"))   
            else:
                logger.error("Could not generate new Envoy configuration: %s" % rc.error)
                logger.error("Raw template output:")
                logger.error("%s" % rc.raw)

            if scout:
                result = scout.report(action="config", result=bool(rc),
                                      runtime=runtime, check=check, generated=(not output_exists))
            else:
                result = {"scout": "inactive"}

        logger.debug("Scout reports %s" % json.dumps(result))

        if 'latest_version' in result:
            latest_semver = get_semver("latest", result['latest_version'])

            # Use scout_version here, not __version__, because the version 
            # coming back from Scout will use build numbers for dev builds,
            # but __version__ 
            current_semver = get_semver("current", scout_version)

            if latest_semver and current_semver:
                logger.debug("Version check: cur %s, latest %s, out of date %s" % 
                             (current_semver, latest_semver, latest_semver > current_semver))

                if latest_semver > current_semver:
                    logger.warning("Upgrade available! to Ambassador version %s" % latest_semver)

        if 'notices' in result:
            for notice in result['notices']:
                try:
                    if isinstance(notice, str):
                        logger.warning(notice)
                    else:
                        lvl = notice['level']
                        msg = notice['message']

                        if isinstance(lvl, str):
                            lvl = getattr(logging, lvl, logging.INFO)

                        logger.log(lvl, "%s", msg)
                except KeyError:
                    logger.warning(json.dumps(notice))
                except TypeError:
                    logger.warning(str(notice))
    except Exception as e:
        handle_exception("EXCEPTION from config", e, 
                         config_dir_path=config_dir_path, output_json_path=output_json_path)

def get_semver(what, version_string):
    semver = None

    try:
        semver = semantic_version.Version(version_string)
    except ValueError:
        logger.warning("Could not perform version check: %s version (%s) is not valid" %
                       (what, version_string))

    return semver

if __name__ == "__main__":
    clize.run([config, dump], alt=[version, showid],
              description="""
              Generate an Envoy config, or manage an Ambassador deployment. Use

              ambassador.py command --help 

              for more help, or 

              ambassador.py --version               

              to see Ambassador's version.
              """)
