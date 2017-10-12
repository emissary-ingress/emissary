#!python

import sys

import logging
import os
import yaml

logging.basicConfig(
    # filename=logPath,
    level=logging.DEBUG, # if appDebug else logging.INFO,
    format="%(asctime)s check-services %(levelname)s: %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S"
)

config_dir = "/etc/ambassador-config"

if len(sys.argv) > 1:
    config_dir = sys.argv[1]

ambassador_yaml_path = os.path.join(config_dir, "ambassador.yaml")
ambassador_yaml = None

try:
    ambassador_yaml = open(ambassador_yaml_path, "r")
except OSError:
    pass

if not ambassador_yaml:
    logging.debug("check-services: %s not present" % ambassador_yaml_path)
    sys.exit(1)

try:
    objects = list(yaml.safe_load_all(ambassador_yaml))
except Exception as e:
    logging.debug("check-services: %s unparseable (%s)" % (ambassador_yaml_path, e))
    sys.exit(1)

ocount = 0
for object in objects:
    ocount += 1
    if ("kind" not in object) or ("name" not in object):
        logging.warning("check-services %s.%d: missing name and/or kind" %
                        (ambassador_yaml_path, ocount))
        continue

    if object["kind"].lower() != "module":
        continue

    if object["name"] != "ambassador":
        continue

    if "config" not in object:
        logging.warning("check-services %s.%d: Ambassador module missing config" %
                        (ambassador_yaml_path, ocount))
        continue

    config = object['config']
    d6e_magic = config.get("datawire-magic", {})

    if d6e_magic.get("demo-services", False):
        logging.debug("check-services %s.%d: demo-services present" % 
                      (ambassador_yaml_path, ocount))
        sys.exit(0)

logging.debug("check-services %s: demo-services not present" % ambassador_yaml_path)
sys.exit(1)

