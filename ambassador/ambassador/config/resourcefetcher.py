from typing import List, Optional, Tuple, TYPE_CHECKING
# from typing import cast as typecast

import json
import logging
import os
import yaml

from .config import Config
from .acresource import ACResource

# Some thoughts:
# - loading a bunch of Ambassador resources is different from loading a bunch of K8s
#   services, because we should assume that if we're being a fed a bunch of Ambassador
#   resources, we'll get a full set. The whole 'secret loader' thing needs to have the
#   concept of a TLSSecret resource that can be force-fed to us, or that can be fetched
#   through the loader if needed.
# - If you're running a debug-loop Ambassador, you should just have a flat (or
#   recursive, I don't care) directory full of Ambassador YAML, including TLSSecrets
#   and Endpoints and whatnot, as needed. All of it will get read by
#   load_from_filesystem and end up in the elements array.
# - If you're running expecting to be fed by kubewatch, at present kubewatch will
#   send over K8s Service records, and anything annotated in there will end up in
#   elements. This may include TLSSecrets or Endpoints. Any TLSSecret mentioned that
#   isn't already in elements will need to be fetched.
# - Ambassador resources do not have namespaces. They have the ambassador_id. That's
#   it. The ambassador_id is completely orthogonal to the namespace. No element with
#   the wrong ambassador_id will end up in elements. It would be nice if they were
#   never sent by kubewatch, but, well, y'know.
# - TLSSecret resources are not TLSContexts. TLSSecrets only have a name, a private
#   half, and a public half. They do _not_ have other TLSContext information.
# - Endpoint resources probably have just a name, a service name, and an endpoint
#   address.

class ResourceFetcher:
    def __init__(self, logger: logging.Logger, aconf: 'Config') -> None:
        self.aconf = aconf
        self.logger = logger
        self.elements: List[ACResource] = []
        self.filename: Optional[str] = None
        self.ocount: int = 1
        self.saved: List[Tuple[Optional[str], int]] = []

    @property
    def location(self):
        return "%s.%d" % (self.filename or "anonymous YAML", self.ocount)

    def push_location(self, filename: Optional[str], ocount: int) -> None:
        self.saved.append((self.filename, self.ocount))
        self.filename = filename
        self.ocount = ocount

    def pop_location(self) -> None:
        self.filename, self.ocount = self.saved.pop()

    def load_from_filesystem(self, config_dir_path, recurse: bool=False, k8s: bool=False):
        inputs: List[Tuple[str, str]] = []

        if os.path.isdir(config_dir_path):
            dirs = [ config_dir_path ]

            while dirs:
                dirpath = dirs.pop(0)

                for filename in os.listdir(dirpath):
                    filepath = os.path.join(dirpath, filename)

                    if recurse and os.path.isdir(filepath):
                        # self.logger.debug("%s: RECURSE" % filepath)
                        dirs.append(filepath)
                        continue

                    if not os.path.isfile(filepath):
                        # self.logger.debug("%s: SKIP non-file" % filepath)
                        continue

                    if not filename.lower().endswith('.yaml'):
                        # self.logger.debug("%s: SKIP non-YAML" % filepath)
                        continue

                    # self.logger.debug("%s: SAVE configuration file" % filepath)
                    inputs.append((filepath, filename))

        else:
            # this allows a file to be passed into the ambassador cli
            # rather than just a directory
            inputs.append((config_dir_path, os.path.basename(config_dir_path)))

        for filepath, filename in inputs:
            self.logger.info("reading %s (%s)" % (filename, filepath))

            try:
                serialization = open(filepath, "r").read()
                self.parse_yaml(serialization, k8s=k8s, filename=filename)
            except IOError as e:
                self.aconf.post_error("could not read YAML from %s: %s" % (filepath, e))

    def parse_yaml(self, serialization: str, k8s=False, rkey: Optional[str]=None,
                   filename: Optional[str]=None) -> None:
        # self.logger.debug("%s: parsing %d byte%s of YAML:\n%s" %
        #                   (self.location, len(serialization), "" if (len(serialization) == 1) else "s",
        #                    serialization))

        try:
            objects = list(yaml.safe_load_all(serialization))

            self.push_location(filename, 1)

            for obj in objects:
                if k8s:
                    self.extract_k8s(obj)
                else:
                    # if not obj:
                    #     self.logger.debug("%s: empty object from %s" % (self.location, serialization))

                    self.process_object(obj, rkey=rkey)
                    self.ocount += 1

            self.pop_location()
        except yaml.error.YAMLError as e:
            self.aconf.post_error("%s: could not parse YAML: %s" % (self.location, e))

    def extract_k8s(self, obj: dict) -> None:
        # self.logger.debug("extract_k8s obj %s" % json.dumps(obj, indent=4, sort_keys=True))

        kind = obj.get('kind', None)
        metadata = obj.get('metadata', None)
        resource_name = metadata.get('name') if metadata else None
        resource_namespace = metadata.get('namespace', 'default') if metadata else None
        resource_identifier = self.filename

        if resource_name and resource_namespace:
            # This resource identifier is useful for log output since filenames can be duplicated (multiple subdirectories)
            resource_identifier = '{name}.{namespace}'.format(namespace=resource_namespace, name=resource_name)
            self.push_location(resource_identifier, 1)

        annotations = metadata.get('annotations', None) if metadata else None

        if annotations:
            annotations = annotations.get('getambassador.io/config', None)
            # self.logger.debug("annotations %s" % annotations)

        skip = False

        if kind != "Service":
            # self.logger.debug("%s: ignoring K8s %s object" % (self.location, kind))
            skip = True

        if not skip and not metadata:
            # self.logger.debug("%s: ignoring unannotated K8s %s" % (self.location, kind))
            skip = True

        if not skip and not resource_name:
            # This should never happen as the name field is required in metadata for Service
            # self.logger.debug("%s: ignoring unnamed K8s %s" % (self.location, kind))
            skip = True

        if not skip and not annotations:
            self.logger.debug("%s: ignoring K8s %s without Ambassador annotation" % (self.location, kind))
            skip = True

        if not skip and (Config.single_namespace and (resource_namespace != self.aconf.ambassador_namespace)):
            # This should never happen in actual usage, since we shouldn't be given things
            # in the wrong namespace. However, in development, this can happen a lot.
            self.logger.debug("%s: ignoring K8s %s in wrong namespace" % (self.location, kind))
            skip = True

        if not skip:
            if not self.filename.endswith(":annotation"):
                self.filename += ":annotation"

            self.parse_yaml(annotations, filename=self.filename, rkey=resource_identifier)

        self.pop_location()

    def process_object(self, obj: dict, rkey: Optional[str]=None) -> None:
        if not isinstance(obj, dict):
            # Bug!!
            if not obj:
                self.aconf.post_error("%s is empty" % self.location)
            else:
                self.aconf.post_error("%s is not a dictionary? %s" %
                                      (self.location, json.dumps(obj, indent=4, sort_keys=4)))
            return

        if not self.aconf.good_ambassador_id(obj):
            # self.logger.debug("%s ignoring K8s Service with mismatched ambassador_id" % self.location)
            return

        if 'kind' not in obj:
            # Bug!!
            self.aconf.post_error("%s is missing 'kind'?? %s" %
                                  (self.location, json.dumps(obj, indent=4, sort_keys=True)))
            return

        # self.logger.debug("%s PROCESS %s initial rkey %s" % (self.location, obj['kind'], rkey))

        # Is this a pragma object?
        if obj['kind'] == 'Pragma':
            # Why did I think this was a good idea? [ :) ]
            new_source = obj.get('source', None)

            if new_source:
                # We don't save the old self.filename here, so this change will last until
                # the next input source (or the next Pragma).
                self.filename = new_source

            # Don't count Pragma objects, since the user generally doesn't write them.
            self.ocount -= 1
            return

        if not rkey:
            rkey = self.filename

        rkey = "%s.%d" % (rkey, self.ocount)

        # self.logger.debug("%s PROCESS %s updated rkey to %s" % (self.location, obj['kind'], rkey))

        # Fine. Fine fine fine.
        serialization = yaml.safe_dump(obj, default_flow_style=False)

        r = ACResource.from_dict(rkey, rkey, serialization, obj)
        self.elements.append(r)

        self.logger.debug("%s PROCESS %s save %s" % (self.location, obj['kind'], rkey))

    def sorted(self, key=lambda x: x.rkey): # returns an iterator, probably
        return sorted(self.elements, key=key)
