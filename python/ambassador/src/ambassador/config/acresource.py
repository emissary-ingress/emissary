# from typing import List
from typing import TYPE_CHECKING, Optional, Type, TypeVar
from typing import cast as typecast

from ..resource import Resource

R = TypeVar("R", bound=Resource)


class ACResource(Resource):
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
      "getambassador.io/v0" if not specified.

    - serialization (keyword-only) is the _original input serialization_, if we have
      it, of the object. If we don't have it, this should be None -- don't just serialize
      the object to no purpose.

    - any additional keyword arguments are saved in the Resource.

    :param rkey: unique identifier for this source, should be short
    :param location: where should a human go to find the source of this resource?
    :param kind: what kind of thing is this?
    :param name: what's the name of this thing?
    :param namespace: what namespace is this in?
    :param metadata_labels: what label did we select to get this thing?
    :param apiVersion: API version, defaults to "getambassador.io/v0" if not present
    :param serialization: original input serialization of obj, if we have it
    :param kwargs: key-value pairs that form the data object for this resource
    """

    name: str
    apiVersion: str

    def __init__(
        self,
        rkey: str,
        location: str,
        *,
        kind: str,
        name: Optional[str] = None,
        namespace: Optional[str] = None,
        metadata_labels: Optional[str] = None,
        apiVersion: Optional[str] = "getambassador.io/v0",
        serialization: Optional[str] = None,
        **kwargs,
    ) -> None:
        if not rkey:
            raise Exception("ACResource requires rkey")

        if not kind:
            raise Exception("ACResource requires kind")

        if (kind != "Pragma") and not name:
            raise Exception("ACResource: %s requires name (%s)" % (kind, repr(kwargs)))

        if not apiVersion:
            raise Exception("ACResource requires apiVersion")

        # This next bit is a little odd -- we don't want a label_selector of None
        # to appear with a null value in the dict, so we move it to kwargs if present
        # and don't bother passing it as an explicit keyword argument.

        if metadata_labels:
            kwargs["metadata_labels"] = metadata_labels

        # print("ACResource __init__ (%s %s)" % (kind, name))

        super().__init__(
            rkey=rkey,
            location=location,
            kind=kind,
            name=name,
            namespace=namespace,
            apiVersion=typecast(str, apiVersion),
            serialization=serialization,
            **kwargs,
        )

    # XXX It kind of offends me that we need this, exactly. Meta-ize this maybe?
    @classmethod
    def from_resource(
        cls: Type[R],
        other: R,
        rkey: Optional[str] = None,
        location: Optional[str] = None,
        kind: Optional[str] = None,
        serialization: Optional[str] = None,
        name: Optional[str] = None,
        namespace: Optional[str] = None,
        metadata_labels: Optional[str] = None,
        apiVersion: Optional[str] = None,
        **kwargs,
    ) -> R:
        new_name = name or other.name
        new_apiVersion = apiVersion or other.apiVersion
        new_namespace = namespace or other.namespace
        new_metadata_labels = metadata_labels or other.get("metadata_labels", None)

        # mypy 0.730 is Just Flat Wrong here. It tries to be "more strict" about
        # super(), which is fine, but it also flags this particular super() call
        # as an error, even though Type[R] is necessarily a Resource type.
        #
        # Since it's complaining about "argument 2 for super is not an instance
        # of argument 1", we need to assert() isinstance here -- but of course,
        # cls is _not_ an instance at all, it's a class, so isinstance() will
        # fail at runtime. So we only do the assertion if TYPE_CHECKING. Grrrr.
        if TYPE_CHECKING:
            assert isinstance(cls, Resource)  # pragma: no cover

        return super().from_resource(
            other,
            rkey=rkey,
            location=location,
            kind=kind,
            name=new_name,
            apiVersion=new_apiVersion,
            namespace=new_namespace,
            metadata_labels=new_metadata_labels,
            serialization=serialization,
            **kwargs,
        )

    # ACResource.INTERNAL is the magic ACResource we use to represent something created by
    # Ambassador's internals.
    @classmethod
    def internal_resource(cls) -> "ACResource":
        return ACResource(
            "--internal--",
            "--internal--",
            kind="Internal",
            name="Ambassador Internals",
            version="getambassador.io/v0",
            description="The '--internal--' source marks objects created by Ambassador's internal logic.",
        )

    # ACResource.DIAGNOSTICS is the magic ACResource we use to represent something created by
    # Ambassador's diagnostics logic. (We could use ACResource.INTERNAL here, but explicitly
    # calling out diagnostics stuff actually helps with, well, diagnostics.)
    @classmethod
    def diagnostics_resource(cls) -> "ACResource":
        return ACResource(
            "--diagnostics--",
            "--diagnostics--",
            kind="Diagnostics",
            name="Ambassador Diagnostics",
            version="getambassador.io/v0",
            description="The '--diagnostics--' source marks objects created by Ambassador to assist with diagnostic output.",
        )
