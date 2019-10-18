import sys

sys.path.append('./python')

import os
import json
import yaml

from Mapping_pb2 import Mapping
from AuthService_pb2 import AuthService
from google.protobuf import json_format

for root, dirs, files in os.walk("../demo/config"):
    for filepath in files:
        path = os.path.join(root, filepath)
        yaml_text = open(path, "r").read()
        objects = yaml.safe_load_all(yaml_text)

        print(f"==== {filepath}:")
        print(f"YAML:\n{yaml_text}")

        for obj in objects:
        	json_text = json.dumps(obj, sort_keys=True, indent=2)
	        print(f"JSON: {json_text}")

	        pbkind = None

	        if obj["kind"] == 'AuthService':
	        	pbkind = AuthService()
	        elif obj["kind"] == 'Mapping':
	        	pbkind = Mapping()

	        if pbkind:
	        	pbobj = json_format.Parse(json_text, pbkind)

	        	print(f"PB object: {pbobj}")

	        	pbjson = json_format.MessageToJson(pbobj)
	        	print(f"PB JSON: {pbjson}")

