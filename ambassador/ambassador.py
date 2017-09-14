import sys

import json
import logging
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

scout_version = "+".join(__version__.split('-', 1))

scout = Scout(app="ambassador", version=scout_version, 
              id_plugin=Scout.configmap_install_id_plugin)

def handle_exception(what, e, **kwargs):
    tb = "\n".join(traceback.format_exception(*sys.exc_info()))

    result = scout.report(action=what, exception=str(e), traceback=tb, **kwargs)

    time.sleep(1)

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

    print("%s" % scout.install_id)

def publish(config_dir_path:Parameter.REQUIRED, output_config_map="ambassador-config"):
    """
    Push an Ambassador configuration to a configmap

    :param config_dir_path: Configuration directory to scan for Ambassador YAML files
    :param output_config_map: Name of map to update
    """

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
            rc = aconf.envoy_config_object()

            if rc:
                aconf.pretty(rc.envoy_config, out=open(output_json_path, "w"))   
            else:
                logging.error("Could not generate new Envoy configuration: %s" % rc.error)
                logging.error("Raw template output:")
                logging.error("%s" % rc.raw)

            result = scout.report(action="config", result=bool(rc), check=check, generated=(not output_exists))

        logging.info("Scout reports %s" % json.dumps(result))
    except Exception as e:
        # scout.report(action="WTFO?")
        handle_exception("EXCEPTION from config", e, 
                         config_dir_path=config_dir_path, output_json_path=output_json_path)

if __name__ == "__main__":
    clize.run([config, publish], alt=[version, showid],
              description="""
              Generate an Envoy config, or manage an Ambassador deployment. Use

              ambassador.py command --help 

              for more help, or 

              ambassador.py --version               

              to see Ambassador's version.
              """)
