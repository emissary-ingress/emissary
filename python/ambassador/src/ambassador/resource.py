import sys
from typing import Any, Dict, Optional, Type, TypeVar

from .cache import Cacheable
from .utils import dump_json, parse_yaml

R = TypeVar("R", bound="Resource")


class Resource(Cacheable):
    """
    A resource that's part of the overall Ambassador configuration world. This is
    the base class for IR resources, Ambassador-config resources, etc.

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

    - serialization (keyword-only) is the _original input serialization_, if we have
    it, of the object. If we don't have it, this should be None -- don't just serialize
    the object to no purpose.

    - any additional keyword arguments are saved in the Resource.

    :param rkey: unique identifier for this source, should be short
    :param location: where should a human go to find the source of this resource?
    :param kind: what kind of thing is this?
    :param serialization: original input serialization of obj, if we have it
    :param kwargs: key-value pairs that form the data object for this resource
    """

    rkey: str
    location: str
    kind: str
    serialization: Optional[str]

    # _errors: List[RichStatus]
    _errored: bool
    _referenced_by: Dict[str, "Resource"]

    def __init__(
        self,
        rkey: str,
        location: str,
        *,
        kind: str,
        serialization: Optional[str] = None,
        **kwargs,
    ) -> None:
        if not rkey:
            raise Exception("Resource requires rkey")

        if not kind:
            raise Exception("Resource requires kind")

        # print("Resource __init__ (%s %s)" % (kind, name))

        super().__init__(
            rkey=rkey,
            location=location,
            kind=kind,
            serialization=serialization,
            # _errors=[],
            _referenced_by={},
            **kwargs,
        )

    def sourced_by(self, other: "Resource"):
        self.rkey = other.rkey
        self.location = other.location

    def referenced_by(self, other: "Resource") -> None:
        # print("%s %s REF BY %s %s" % (self.kind, self.name, other.kind, other.rkey))
        self._referenced_by[other.location] = other

    def is_referenced_by(self, other_location) -> Optional["Resource"]:
        return self._referenced_by.get(other_location, None)

    def __getattr__(self, key: str) -> Any:
        try:
            return self[key]
        except KeyError:
            raise AttributeError(key)

    def __setattr__(self, key: str, value: Any) -> None:
        self[key] = value

    def __str__(self) -> str:
        return "<%s %s>" % (self.kind, self.rkey)

    def as_dict(self) -> Dict[str, Any]:
        ad = dict(self)

        ad.pop("rkey", None)
        ad.pop("serialization", None)
        ad.pop("location", None)
        ad.pop("_referenced_by", None)
        ad.pop("_errored", None)

        return ad

    def as_json(self):
        return dump_json(self.as_dict(), pretty=True)

    @classmethod
    def from_resource(
        cls: Type[R],
        other: R,
        rkey: Optional[str] = None,
        location: Optional[str] = None,
        kind: Optional[str] = None,
        serialization: Optional[str] = None,
        **kwargs,
    ) -> R:
        """
        Create a Resource by copying another Resource, possibly overriding elements
        along the way.

        NOTE WELL: if you pass in kwargs, we assume that any values are safe to use as-is
        and DO NOT COPY THEM. Otherwise, we SHALLOW COPY other.attrs for the new Resource.

        :param other: the base Resource we're copying
        :param rkey: optional new rkey
        :param location: optional new location
        :param kind: optional new kind
        :param serialization: optional new original input serialization
        :param kwargs: optional new key-value pairs -- see discussion about copying above!
        """

        # rkey and location are required positional arguments. Fine.
        new_rkey = rkey or other.rkey
        new_location = location or other.location

        # Make a shallow-copied dict that we can muck with...
        new_attrs = dict(kwargs) if kwargs else dict(other)

        # Don't include kind unless it comes in on this call.
        if kind:
            new_attrs["kind"] = kind
        else:
            new_attrs.pop("kind", None)

        # Don't include serialization at all if we don't have one.
        if serialization:
            new_attrs["serialization"] = serialization
        elif other.serialization:
            new_attrs["serialization"] = other.serialization

        # Make sure that things that shouldn't propagate are gone...
        new_attrs.pop("rkey", None)
        new_attrs.pop("location", None)
        new_attrs.pop("_errors", None)
        new_attrs.pop("_errored", None)
        new_attrs.pop("_referenced_by", None)

        # ...and finally, use new_attrs for all the keyword args when we set up
        # the new instance.
        return cls(new_rkey, new_location, **new_attrs)

    @classmethod
    def from_dict(
        cls: Type[R],
        rkey: str,
        location: str,
        serialization: Optional[str],
        attrs: Dict,
    ) -> R:
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
        ambassador = sys.modules["ambassador"]

        resource_class: Optional[Type[R]] = getattr(ambassador, attrs["kind"], None)

        if not resource_class:
            resource_class = getattr(ambassador, "AC" + attrs["kind"], cls)
        assert resource_class

        # print("%s.from_dict: %s => %s" % (cls, attrs['kind'], resource_class))

        return resource_class(
            rkey, location=location, serialization=serialization, **attrs
        )

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

        attrs = parse_yaml(serialization)

        return cls.from_dict(rkey, location, serialization, attrs)
