from typing import List, Optional, Type, TypeVar
from typing import cast as typecast

import os
import yaml

from ..resource import Resource

R = TypeVar('R', bound=Resource)


class ACResource (Resource):
    """
    A resource that we're going to use as part of the Ambassador configuration.

    Elements in a Resource:
    - rkey is a short identifier that is used as the primary key for _all_ the
    Ambassador classes to identify this single specific resource. It should be
    something like "ambassador-default.1" or the like: very specific, doesn't
    have to be fun for humans.

    - location is a more human-readable string describing where the human should
    go to find the source of this resource. "Service ambassador, namespace default,
    object 1". This isn't really used by the Config class, but the Diagnostics class
    makes heavy use of it.

    - kind (keyword-only) is what kind of Ambassador resource this is.

    - name (keyword-only) is the name of the Ambassador resource.

    - apiVersion (keyword-only) specifies the API version in use. It defaults to
    "ambassador/v0" if not specified.

    - serialization (keyword-only) is the _original input serialization_, if we have
    it, of the object. If we don't have it, this should be None -- don't just serialize
    the object to no purpose.

    - any additional keyword arguments are saved in the Resource.

    :param rkey: unique identifier for this source, should be short
    :param location: where should a human go to find the source of this resource?
    :param kind: what kind of thing is this?
    :param name: what's the name of this thing?
    :param apiVersion: API version, defaults to "ambassador/v0" if not present
    :param serialization: original input serialization of obj, if we have it
    :param kwargs: key-value pairs that form the data object for this resource
    """

    name: str
    apiVersion: str

    def __init__(self, rkey: str, location: str, *,
                 kind: str,
                 name: Optional[str]=None,
                 apiVersion: Optional[str]="ambassador/v0",
                 serialization: Optional[str]=None,
                 **kwargs) -> None:

        if not rkey:
            raise Exception("ACResource requires rkey")

        if not kind:
            raise Exception("ACResource requires kind")

        if (kind != "Pragma") and not name:
            raise Exception("ACResource: %s requires name (%s)" % (kind, repr(kwargs)))

        if not apiVersion:
            raise Exception("ACResource requires apiVersion")

        # print("ACResource __init__ (%s %s)" % (kind, name))

        super().__init__(rkey=rkey, location=location,
                         kind=kind, name=name,
                         apiVersion=typecast(str, apiVersion),
                         serialization=serialization,
                         **kwargs)

    # XXX It kind of offends me that we need this, exactly. Meta-ize this maybe?
    @classmethod
    def from_resource(cls: Type[R], other: R,
                      rkey: Optional[str]=None,
                      location: Optional[str]=None,
                      kind: Optional[str]=None,
                      serialization: Optional[str]=None,
                      name: Optional[str]=None,
                      apiVersion: Optional[str]=None,
                      **kwargs) -> R:
        new_name = name or other.name
        new_apiVersion = apiVersion or other.apiVersion

        return super().from_resource(other, rkey=rkey, location=location, kind=kind,
                                     name=new_name, apiVersion=new_apiVersion,
                                     serialization=serialization, **kwargs)

    # ACResource.INTERNAL is the magic ACResource we use to represent something created by
    # Ambassador's internals.
    @classmethod
    def internal_resource(cls) -> 'ACResource':
        return ACResource(
            "--internal--", "--internal--",
            kind="Internal",
            name="Ambassador Internals",
            version="ambassador/v0",
            description="The '--internal--' source marks objects created by Ambassador's internal logic."
        )

    # ACResource.DIAGNOSTICS is the magic ACResource we use to represent something created by
    # Ambassador's diagnostics logic. (We could use ACResource.INTERNAL here, but explicitly
    # calling out diagnostics stuff actually helps with, well, diagnostics.)
    @classmethod
    def diagnostics_resource(cls) -> 'ACResource':
        return ACResource(
            "--diagnostics--", "--diagnostics--",
            kind="Diagnostics",
            name="Ambassador Diagnostics",
            version="ambassador/v0",
            description="The '--diagnostics--' source marks objects created by Ambassador to assist with diagnostic output."
        )


########
## ResourceFetcher and fetch_resources are the Canonical Way to load ambassador config
## resources from disk.


class ResourceFetcher:
    def __init__(self, config_dir_path: str, logger, k8s: bool=False) -> None:
        self.logger = logger
        self.resources: List[ACResource] = []

        inputs = []

        if os.path.isdir(config_dir_path):
            for filename in os.listdir(config_dir_path):
                filepath = os.path.join(config_dir_path, filename)

                if not os.path.isfile(filepath):
                    continue
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

            serialization = open(filepath, "r").read()

            self.load_yaml(serialization, k8s=k8s)

            # self.logger.debug("%s: parsed ocount %d" % (self.filename, self.ocount))

            self.filename = None
            self.filepath = None
            self.ocount = 0

    def load_yaml(self, serialization: str, rkey: Optional[str]=None, k8s: bool=False) -> None:
        objects = list(yaml.safe_load_all(serialization))

        for obj in objects:
            if k8s:
                self.extract_k8s(obj)
                self.ocount += 1
            else:
                self.ocount = self.process_object(obj, rkey)

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

        self.filename += ":annotation"
        self.load_yaml(annotations, rkey=resource_identifier)

    def process_object(self, obj: dict, rkey: Optional[str]=None, k8s: bool=False) -> int:
        # self.logger.debug("%s.%d PROCESS %s" % (self.filename, self.ocount, obj['kind']))

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


def fetch_resources(config_dir_path: str, logger, k8s=False):
    fetcher = ResourceFetcher(config_dir_path, logger, k8s=k8s)
    return fetcher.__iter__()
