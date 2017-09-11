import sys

import click
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
@click.argument('output_json_path', type=click.Path(exists=True))
#help="Path to which to write Envoy configuration")
def generate_envoy_json(check, config_dir_path, output_json_path):
    logging.info("CHECK MODE  %s" % check)
    logging.info("CONFIG DIR  %s" % config_dir_path)
    logging.info("OUTPUT PATH %s" % output_json_path)

    aconf = AmbassadorConfig(config_dir_path)
    econf = aconf.envoy_config_object()

    aconf.pretty(econf, out=open(output_json_path, "w"))   

if __name__ == "__main__":
    generate_envoy_json()
