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

from .config import Config
from .utils import RichStatus

from .VERSION import Version

__version__ = Version

logging.basicConfig(
    level=logging.INFO, # if appDebug else logging.INFO,
    format="%%(asctime)s ambassador %s %%(levelname)s: %%(message)s" % __version__,
    datefmt="%Y-%m-%d %H:%M:%S"
)

# logging.getLogger("datawire.scout").setLevel(logging.DEBUG)
logger = logging.getLogger("ambassador")
logger.setLevel(logging.DEBUG)

def handle_exception(what, e, **kwargs):
    tb = "\n".join(traceback.format_exception(*sys.exc_info()))

    if Config.scout:
        result = Config.scout_report(action=what, mode="cli", exception=str(e), traceback=tb,
                                     runtime=Config.runtime, **kwargs)

        logger.debug("Scout %s, result: %s" %
                     ("disabled" if Config.scout.disabled else "enabled", result))

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

    if Config.scout:
        print("%s" % Config.scout.install_id)
    else:
        print("unknown")

def parse_config(config_dir_path, template_dir_path=None, schema_dir_path=None):
    try:
        return Config(config_dir_path, 
                      template_dir_path=template_dir_path, schema_dir_path=schema_dir_path)
    except Exception as e:
        handle_exception("EXCEPTION from parse_config", e, 
                         config_dir_path=config_dir_path, template_dir_path=template_dir_path,
                         schema_dir_path=schema_dir_path)

        # This is fatal.
        sys.exit(1)

def dump(config_dir_path:Parameter.REQUIRED):
    """
    Dump the intermediate form of an Ambassador configuration for debugging

    :param config_dir_path: Configuration directory to scan for Ambassador YAML files
    """

    try:
        aconf = parse_config(config_dir_path)
        json.dump(aconf.envoy_config, sys.stdout, indent=4, sort_keys=True)
    except Exception as e:
        handle_exception("EXCEPTION from dump", e, 
                         config_dir_path=config_dir_path)

        # This is fatal.
        sys.exit(1)

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

        rc = RichStatus.fromError("impossible error")

        if not output_exists:
            # Either we didn't need to check, or the check didn't turn up
            # a valid config. Regenerate.
            logger.info("Generating new Envoy configuration...")
            aconf = parse_config(config_dir_path)
            rc = aconf.generate_envoy_config(mode="cli", check=check)

            if rc:
                aconf.pretty(rc.envoy_config, out=open(output_json_path, "w"))   
            else:
                logger.error("Could not generate new Envoy configuration: %s" % rc.error)
                logger.error("Raw template output:")
                logger.error("%s" % rc.raw)
        else:
            result = Config.scout_report(action="config", result=True,
                                         mode="cli", generated=False, check=check)

            rc = RichStatus.OK(scout_result=result)
        
        if 'latest_version' in rc.scout_result:
            latest_semver = get_semver("latest", rc.scout_result['latest_version'])

            # Use Config.scout_version here, not __version__, because the version
            # coming back from Scout will use build numbers for dev builds, but
            # __version__ won't.
            current_semver = get_semver("current", Config.scout_version)

            if latest_semver and current_semver:
                logger.debug("Version check: cur %s, latest %s, out of date %s" % 
                             (current_semver, latest_semver, latest_semver > current_semver))

                if latest_semver > current_semver:
                    logger.warning("Upgrade available! to Ambassador version %s" % latest_semver)

        if 'notices' in rc.scout_result:
            for notice in rc.scout_result['notices']:
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

        # This is fatal.
        sys.exit(1)

def get_semver(what, version_string):
    semver = None

    try:
        semver = semantic_version.Version(version_string)
    except ValueError:
        logger.warning("Could not perform version check: %s version (%s) is not valid" %
                       (what, version_string))

    return semver

def main():
    clize.run([config, dump], alt=[version, showid],
              description="""
              Generate an Envoy config, or manage an Ambassador deployment. Use

              ambassador.py command --help 

              for more help, or 

              ambassador.py --version               

              to see Ambassador's version.
              """)

if __name__ == "__main__":
    main()
