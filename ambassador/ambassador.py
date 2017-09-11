import sys

import click
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

logging.info("DECLARATIVE! running")

@click.command()
@click.option('--check', default=False, is_flag=True,
              help="Only regenerate output if it doesn't exist")
@click.argument('config_dir_path', type=click.Path(exists=True))
#help="Path of directory to scan for configuration info")
@click.argument('output_json_path', type=click.Path(readable=False))
#help="Path to which to write Envoy configuration")
def generate_envoy_json(check, config_dir_path, output_json_path):
    logging.info("CHECK MODE  %s" % check)
    logging.info("CONFIG DIR  %s" % config_dir_path)
    logging.info("OUTPUT PATH %s" % output_json_path)

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
    generate_envoy_json()
