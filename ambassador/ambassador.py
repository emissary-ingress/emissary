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

if __name__ == "__main__":
    config_dir_path = sys.argv[1]
    output_json_path = sys.argv[2]

    aconf = AmbassadorConfig(config_dir_path)
    econf = aconf.envoy_config_object()

    aconf.pretty(econf, out=open(output_json_path, "w"))

