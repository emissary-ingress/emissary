import sys

import clize
from clize import Parameter

import json
import logging

from AmbassadorConfig import AmbassadorConfig

import VERSION

__version__ = VERSION.Version

logging.basicConfig(
    level=logging.DEBUG, # if appDebug else logging.INFO,
    format="%%(asctime)s ambassador %s %%(levelname)s: %%(message)s" % __version__,
    datefmt="%Y-%m-%d %H:%M:%S"
)

def version():
    """
    Show Ambassador's version
    """

    print("Ambassador %s" % __version__)

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

    logging.debug("CHECK MODE  %s" % check)
    logging.debug("CONFIG DIR  %s" % config_dir_path)
    logging.debug("OUTPUT PATH %s" % output_json_path)

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
            logging.debug("output file does not exist")
            output_exists = False
        except OSError:
            logging.warning("output file is not sane?")
            output_exists = False
        except json.decoder.JSONDecodeError:
            logging.warning("output file is not valid JSON")
            output_exists = False

        logging.info("Output file %s" % ("exists" if output_exists else "does not exist"))

    if not output_exists:
        # Either we didn't need to check, or the check didn't turn up
        # a valid config. Regenerate.
        logging.info("Generating new Envoy configuration...")
        aconf = AmbassadorConfig(config_dir_path)
        econf = aconf.envoy_config_object()

        aconf.pretty(econf, out=open(output_json_path, "w"))   

if __name__ == "__main__":
    clize.run([config, publish], alt=[version],
              description="""
              Generate an Envoy config, or manage an Ambassador deployment. Use

              ambassador.py command --help 

              for more help, or 

              ambassador.py --version               

              to see Ambassador's version.
              """)
