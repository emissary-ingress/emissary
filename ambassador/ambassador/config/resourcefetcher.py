import json
import os
import yaml

from typing import List, Optional, TYPE_CHECKING
# from typing import Type, TypeVar
# from typing import cast as typecast

from .acresource import ACResource
from ..utils import RichStatus

if TYPE_CHECKING:
    from .config import Config

########
## ResourceFetcher and fetch_resources are the Canonical Way to load ambassador config
## resources from disk.


class ResourceFetcher:
    def __init__(self, aconf: 'Config', config_dir_path: str, k8s: bool=False) -> None:
        self.aconf = aconf
        self.logger = aconf.logger
        self.resources: List[ACResource] = []

        inputs = []

        if os.path.isdir(config_dir_path):
            for filename in os.listdir(config_dir_path):
                filepath = os.path.join(config_dir_path, filename)

                if not filename.lower().endswith('.yaml'):
                    self.logger.debug("%s: SKIP non-YAML" % filepath)
                    continue

                if not os.path.isfile(filepath):
                    self.logger.debug("%s: SKIP non-file" % filepath)
                    continue

                self.logger.debug("%s: SAVE configuration file" % filepath)
                inputs.append((filepath, filename))
        else:
            # this allows a file to be passed into the ambassador cli
            # rather than just a directory
            inputs.append((config_dir_path, os.path.basename(config_dir_path)))

        for filepath, filename in inputs:
            self.filename: Optional[str] = filename
            self.filepath: Optional[str] = filepath
            self.ocount: int = 1

            # self.logger.debug("%s: init ocount %d" % (self.filename, self.ocount))

            try:
                serialization = open(filepath, "r").read()
            except IOError as e:
                self.post_error(RichStatus.fromError("%s: could not load YAML: %s" % (filepath, e)),
                                unparsed_resource=True)
                continue

            self.load_yaml(serialization, k8s=k8s)

            # self.logger.debug("%s: parsed ocount %d" % (self.filename, self.ocount))

            self.filename = None
            self.filepath = None
            self.ocount = 0

    def post_error(self, rc: RichStatus, resource: ACResource=None, unparsed_resource=False):
        self.aconf.post_error(rc, resource=resource, unparsed_resource=unparsed_resource)

    def load_yaml(self, serialization: str, rkey: Optional[str]=None, k8s: bool=False) -> None:
        try:
            objects = list(yaml.safe_load_all(serialization))

            for obj in objects:
                if k8s:
                    self.extract_k8s(obj)
                    self.ocount += 1
                else:
                    self.ocount = self.process_object(obj, rkey)
        except yaml.error.YAMLError as e:
            self.post_error(RichStatus.fromError("%s: could not parse YAML: %s" % (filepath, e)),
                            unparsed_resource=True)

    def extract_k8s(self, obj: dict) -> None:
        kind = obj.get('kind', None)

        if kind != "Service":
            self.logger.debug("%s.%s: ignoring K8s %s object" % (self.filepath, self.ocount, kind))
            return

        metadata = obj.get('metadata', None)

        if not metadata:
            self.logger.debug("%s.%s: ignoring unannotated K8s %s" % (self.filepath, self.ocount, kind))
            return

        # Use metadata to build a unique resource identifier
        resource_name = metadata.get('name')

        # This should never happen as the name field is required in metadata for Service
        if not resource_name:
            self.logger.debug("%s.%s: ignoring unnamed K8s %s" % (self.filepath, self.ocount, kind))
            return

        resource_namespace = metadata.get('namespace', 'default')

        # This resource identifier is useful for log output since filenames can be duplicated (multiple subdirectories)
        resource_identifier = '{name}.{namespace}'.format(namespace=resource_namespace, name=resource_name)

        annotations = metadata.get('annotations', None)

        if annotations:
            annotations = annotations.get('getambassador.io/config', None)

        # self.logger.debug("annotations %s" % annotations)

        if not annotations:
            self.logger.debug("%s.%s: ignoring K8s %s without Ambassador annotation" %
                              (self.filepath, self.ocount, kind))
            return

        if self.filename and (not self.filename.endswith(":annotation")):
            self.filename += ":annotation"

        self.load_yaml(annotations, rkey=resource_identifier)

    def process_object(self, obj: dict, rkey: Optional[str]=None, k8s: bool=False) -> int:
        # self.logger.debug("%s.%d PROCESS %s" % (self.filename, self.ocount, obj['kind']))

        if not isinstance(obj, dict):
            # Bug!!
            self.post_error(RichStatus.fromError("%s.%d is not a dictionary? %s" %
                                                 (self.filename, self.ocount, json.dumps(obj, indent=4, sort_keys=4))),
                            unparsed_resource=True)
            return self.ocount + 1

        if 'kind' not in obj:
            # Bug!!
            self.post_error(RichStatus.fromError("%s.%d is missing 'kind'?? %s" %
                                                 (self.filename, self.ocount, json.dumps(obj, indent=4, sort_keys=4))),
                            unparsed_resource=True)
            return self.ocount + 1

        # Is this a pragma object?
        if obj['kind'] == 'Pragma':
            # Yes. Handle this inline and be done.
            keylist = sorted([ x for x in sorted(obj.keys()) if ((x != 'apiVersion') and (x != 'kind')) ])

            # self.logger.debug("PRAGMA %s" % ", ".join(keylist))

            for key in keylist:
                if key == 'source':
                    self.filename = obj['source']

                    self.logger.debug("PRAGMA: override source_name to %s" % self.filename)

            return self.ocount

        # Not a pragma.

        if not rkey:
            rkey = "%s.%d" % (self.filename, self.ocount)
        elif rkey and k8s:
            # rkey should be unique for k8s objects
            rkey = (''.join([rkey, ".%d" % self.ocount]))

        # Fine. Fine fine fine.
        serialization = yaml.safe_dump(obj, default_flow_style=False)

        r = ACResource.from_dict(rkey, rkey, serialization, obj)
        self.resources.append(r)

        self.logger.debug("%s.%d: save %s %s" %
                          (self.filename, self.ocount, obj['kind'], obj['name']))

        return self.ocount + 1

    def __iter__(self):
        return self.resources.__iter__()
