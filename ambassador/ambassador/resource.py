from typing import Any, ClassVar, Dict, Iterable, List, Optional, Tuple, Union
# from typing import cast as typecast

import yaml

# from .utils import RichStatus, SourcedDict
# from .mapping import Mapping

class Resource:
    def __init__(self, res_key: str, location: str, attrs: Dict[str, Any], serialization: Optional[str]=None) -> None:
        """
        A resource that we're going to use as part of the Ambassador configuration.

        Elements in a Resource:
        - res_key is a short identifier that is used as the primary key for _all_ the
        Ambassador classes to identify this single specific resource. It should be
        something like "ambassador-default.1" or the like: very specific, doesn't
        have to be fun for humans.

        - location is a more human-readable string describing where the human should
        go to find the source of this resource. "Service ambassador, namespace default,
        object 1". This isn't really used by the Config class, but the Diagnostics class
        makes heavy use of it.

        - obj is the actual object contained by this resource. It must contain
        'apiVersion' and 'kind' elements like a good resource.

        - serialization is the _original input serialization_, if we have it, of the
        object. If we don't have it, this should be None -- don't just serialize the
        object to no purpose.

        :param res_key: unique identifier for this source, should be short
        :param location: where should a human go to find the source of this resource?
        :param attrs: dictionary containing the data object for this resource
        :param serialization: original input serialization of obj, if we have it
        """

        self.res_key = res_key
        self.location = location
        self.serialization = serialization
        self.attrs = attrs

        self.errors: List[str] = []

        self.kind: str = self._look_for_attr('kind')
        self.version: str = self._look_for_attr('apiVersion')
        self.name: str = self._look_for_attr('name')

    def _look_for_attr(self, key: str) -> str:
        if key in self.attrs:
            return self[key]
        else:
            self.post_error('missing attribute: %s' % key)
            return '<no %s>' % key

    def post_error(self, error: str):
        self.errors.append(error)

    def __getitem__(self, key: str) -> Any:
        """
        Allow resource[key] to look into the attributes object directly.

        :param key: key to look up
        :return: contents of self.attrs[key]
        """
        return self.attrs[key]

    def get(self, key: str, *args) -> Any:
        """
        Allow resource.get(key, default) to work.

        get is a little weird because supplying a default of None is different
        from supplying no default at all!

        :param key: key to look up
        :param args: default value, if present
        :return: contents of self.attrs[key]
        """

        if len(args) > 0:
            return self.attrs.get(key, args[0])
        else:
            return self.attrs.get(key)

    def __str__(self) -> str:
        return("<%s: %s>" % (self.res_key, self.kind))

    @classmethod
    def from_resource(
            cls,
            other: 'Resource',
            res_key: Optional[str]=None,
            location: Optional[str]=None,
            attrs: Optional[Dict[str, Any]]=None,
            serialization: Optional[str]=None
        ) -> 'Resource':
        """
        Create a Resource by copying another Resource, possibly overriding elements
        along the way.

        NOTE WELL: if you pass in attrs, we assume that your attrs is safe to use as-is
        and DO NOT COPY IT. Otherwise, we SHALLOW COPY other.obj for the new Resource.

        :param other: the base Resource we're copying
        :param res_key: optional new res_key
        :param location: optional new location
        :param attrs: optional new attrs -- see discussion about copying above!
        :param serialization: optional new original input serialization
        """

        new_res_key = res_key if res_key else other.res_key
        new_location = location if location else other.location
        new_serialization = serialization if serialization else other.serialization

        # Since other.attrs is not Optional this is kind of rank paranoia.
        new_attrs = attrs if attrs else dict(other.attrs if other.attrs else {})

        return Resource(new_res_key, new_location, new_attrs, new_serialization)

    @classmethod
    def from_yaml(klass, res_key: str, location: str, serialization: str) -> 'Resource':
        """
        Create a Resource from a YAML serialization. The new Resource's res_key
        and location must be handed in explicitly, and of course in this case the
        serialization is mandatory.

        Raises an exception if the serialization is not parseable.

        :param res_key: unique identifier for this source, should be short
        :param location: where should a human go to find the source of this resource?
        :param serialization: original input serialization of obj
        """

        attrs = yaml.safe_load(serialization)
        return Resource(res_key, location, attrs, serialization)

    # Resource.INTERNAL is the magic Resource we use to represent something created by
    # Ambassador's internals.
    @classmethod
    def internal_resource(cls) -> 'Resource':
        return cls.from_yaml(
            "--internal--",
            "--internal--",
            """
            apiVersion: ambassador/v0
            kind: Internal
            name: Ambassador Internals
            description: The --internal-- source marks objects created by Ambassador's internal logic.
            """
        )

    # Resource.DIAGNOSTICS is the magic Resource we use to represent something created by
    # Ambassador's diagnostics logic. (We could use Resource.INTERNAL here, but explicitly
    # calling out diagnostics stuff actually helps with, well, diagnostics.)
    @classmethod
    def diagnostics_resource(cls) -> 'Resource':
        return cls.from_yaml(
            "--diagnostics--",
            "--diagnostics--",
            """
            apiVersion: ambassador/v0
            kind: Diagnostics
            name: Ambassador Diagnostics
            description: The --diagnostics-- source marks objects created by Ambassador to assist with diagnostics.
            """
        )
