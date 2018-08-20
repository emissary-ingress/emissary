import sys

from typing import Any, Dict, List, Optional, Type, TypeVar
from typing import cast as typecast

import yaml

from .utils import RichStatus

R = TypeVar('R', bound='Resource')


class Resource (dict):
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

    rkey: str
    location: str
    kind: str
    name: str
    apiVersion: str
    serialization: Optional[str]

    _errors: List[RichStatus]
    _referenced_by: Dict[str, 'Resource']

    def __init__(self, rkey: str, location: str, *,
                 kind: str,
                 name: str,
                 apiVersion: Optional[str]="ambassador/v0",
                 serialization: Optional[str]=None,
                 **kwargs) -> None:

        if not rkey:
            raise Exception("Resource requires rkey")

        if not kind:
            raise Exception("Resource requires kind")

        if (kind != "Pragma") and not name:
            raise Exception("Resource requires name")

        if not apiVersion:
            raise Exception("Resource requires apiVersion")

        # print("Resource __init__ (%s %s)" % (kind, name))

        super().__init__(rkey=rkey, location=location,
                         kind=kind, name=name,
                         apiVersion=typecast(str, apiVersion),
                         serialization=serialization,
                         _errors=[],
                         _referenced_by={},
                         **kwargs)

    def references(self, other: 'Resource'):
        """
        Mark another Resource as referenced by this one.

        :param other:
        :return:
        """

        other.referenced_by(self)

    def referenced_by(self, other: 'Resource') -> None:
        # print("%s %s REF BY %s %s" % (self.kind, self.name, other.kind, other.rkey))
        self._referenced_by[other.rkey] = other

    def is_referenced_by(self, other_rkey) -> Optional['Resource']:
        return self._referenced_by.get(other_rkey, None)

    def post_error(self, error: RichStatus):
        self._errors.append(error)

    def __getattr__(self, key: str) -> Any:
        return self[key]

    def __setattr__(self, key: str, value: Any) -> None:
        self[key] = value

    def __str__(self) -> str:
        return("<%s %s>" % (self.kind, self.rkey))

    def as_dict(self) -> Dict[str, Any]:
        ad = dict(self)

        ad.pop('rkey', None)
        ad.pop('serialization', None)
        ad.pop('location', None)
        ad.pop('_referenced_by', None)
        ad.pop('_errors', None)

        return ad

    @classmethod
    def from_resource(cls: Type[R], other: R,
                      rkey: Optional[str]=None,
                      location: Optional[str]=None,
                      kind: Optional[str]=None,
                      name: Optional[str]=None,
                      apiVersion: Optional[str]=None,
                      serialization: Optional[str]=None,
                      **kwargs) -> R:
        """
        Create a Resource by copying another Resource, possibly overriding elements
        along the way.

        NOTE WELL: if you pass in kwargs, we assume that any values are safe to use as-is
        and DO NOT COPY THEM. Otherwise, we SHALLOW COPY other.attrs for the new Resource.

        :param other: the base Resource we're copying
        :param rkey: optional new rkey
        :param location: optional new location
        :param kind: optional new kind
        :param name: optional new name
        :param apiVersion: optional new API version
        :param serialization: optional new original input serialization
        :param kwargs: optional new key-value pairs -- see discussion about copying above!
        """

        # rkey and location are required positional arguments. Fine.
        new_rkey = rkey or other.rkey
        new_location = location or other.location

        # Make a shallow-copied dict that we can muck with...
        new_attrs = dict(kwargs) if kwargs else dict(other)

        # Don't include kind or location unless it comes in on this call.
        if kind:
            new_attrs['kind'] = kind
        else:
            new_attrs.pop('kind', None)

        # Don't include serialization at all if we don't have one.
        if serialization:
            new_attrs['serialization'] = serialization
        elif other.serialization:
            new_attrs['serialization'] = other.serialization

        # After that, stuff other things that need propagation into it...
        new_attrs['name'] = name or other.name
        new_attrs['apiVersion'] = apiVersion or other.apiVersion

        # ...then make sure things that shouldn't propagate are gone.
        new_attrs.pop('rkey', None)
        new_attrs.pop('location', None)
        new_attrs.pop('_errors', None)
        new_attrs.pop('_referenced_by', None)

        # Finally, use new_attrs for all the keyword args when we set up the new instance.
        return cls(new_rkey, new_location, **new_attrs)

    @classmethod
    def from_dict(cls: Type[R], rkey: str, location: str, serialization: Optional[str], attrs: Dict) -> R:
        """
        Create a Resource or subclass thereof from a dictionary. The new Resource's rkey
        and location must be handed in explicitly.

        The difference between this and simply intializing a Resource object is that
        from_dict will introspect the attrs passed in and create whatever kind of Resource
        matches attrs['kind'] -- so for example, if kind is "Mapping", this method will
        return a Mapping rather than a Resource.

        :param rkey: unique identifier for this source, should be short
        :param location: where should a human go to find the source of this resource?
        :param serialization: original input serialization of obj
        :param attrs: dictionary from which to initialize the new object
        """

        # So this is a touch odd but here we go. We want to use the Kind here to find
        # the correct type.
        ambassador = sys.modules['ambassador']
        resource_class: Type[R] = getattr(ambassador, attrs['kind'], cls)

        return resource_class(rkey, location=location, serialization=serialization, **attrs)

    @classmethod
    def from_yaml(cls: Type[R], rkey: str, location: str, serialization: str) -> R:
        """
        Create a Resource from a YAML serialization. The new Resource's rkey
        and location must be handed in explicitly, and of course in this case the
        serialization is mandatory.

        Raises an exception if the serialization is not parseable.

        :param rkey: unique identifier for this source, should be short
        :param location: where should a human go to find the source of this resource?
        :param serialization: original input serialization of obj
        """

        attrs = yaml.safe_load(serialization)

        return cls.from_dict(rkey, location, serialization, attrs)

    # Resource.INTERNAL is the magic Resource we use to represent something created by
    # Ambassador's internals.
    @classmethod
    def internal_resource(cls) -> 'Resource':
        return Resource(
            "--internal--", "--internal--",
            kind="Internal",
            name="Ambassador Internals",
            version="ambassador/v0",
            description="The --internal-- source marks objects created by Ambassador's internal logic."
        )

    # Resource.DIAGNOSTICS is the magic Resource we use to represent something created by
    # Ambassador's diagnostics logic. (We could use Resource.INTERNAL here, but explicitly
    # calling out diagnostics stuff actually helps with, well, diagnostics.)
    @classmethod
    def diagnostics_resource(cls) -> 'Resource':
        return Resource(
            "--diagnostics--", "--diagnostics--",
            kind="Diagnostics",
            name="Ambassador Diagnostics",
            version="ambassador/v0",
            description="The --diagnostics-- source marks objects created by Ambassador to assist with diagnostics."
        )
