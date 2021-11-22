Ambassador Developer's Guide
============================

Concepts
--------

Ambassador sits between users and an Envoy. The primary job that an Ambassador does is to take an _Ambassador configuration_ and, from that, generate an _Envoy configuration_. This generation happens using an _intermediate representation (IR)_ to manage all the internal logic that Ambassador needs:

```Ambassador config => IR => Envoy config```

### Ambassador Components and Ports

Ambassador comprises several different components:

| Component                 | Type   | Function.                 |
| :------------------------ | :----  | :------------------------ |
| `diagd`                   | Python | Increasingly-misnamed core of the system; manages changing configurations and provides the diagnostics UI |
| `ambex`                   | Go     | Envoy `go-control-plane` implementation; supplies Envoy with current configuration |
| `watt`                    | Go     | Service/secret discovery; interface to Kubernetes and Consul |
| `envoy`                   | C++    | The actual proxy process |
| `kubewatch.py`            | Python | Used only to determine the cluster's installation ID; needs to be subsumed by `watt` |

`diagd`, `ambex`, `watt`, and `envoy` are all long-running daemons. If any of them exit, the pod as a whole will exit.

Ambassador uses several TCP ports while running. All but one of them are in the range 8000-8499, and any future assignments for Ambassador ports should come from this range.

| Port | Process | Function |
| :--- | :------ | :------- |
| 8001 | `envoy` | Internal stats, logging, etc.; not exposed outside pod |
| 8002 | `watt`  | Internal `watt` snapshot access; not exposed outside pod |
| 8003 | `ambex` | Internal `ambex` snapshot access; not exposed outside pod |
| 8004 | `diagd` | Internal `diagd` access when `AMBASSADOR_FAST_RECONFIGURE` is set; not exposed outside pod |
| 8043 | `entrypoint` | CR Conversion API exposed by `entrypoint` |
| 8080 | `envoy` | Default HTTP service port |
| 8443 | `envoy` | Default HTTPS service port |
| 8877 | `diagd` | Direct access to diagnostics UI; provided by `busyambassador entrypoint` when `AMBASSADOR_FAST_RECONFIGURE` is set |

### The Ambassador Configuration

An Ambassador configuration is a collection of _Ambassador configuration resources_, which are represented by subclasses of `ambassador.config.ACResource`. The configuration as a whole is represented by an `ambassador.Config` object.

`ambassador.Config` does not know how to parse YAML, interact with Kubernetes, or look at the filesystem. Instead, its consumer must construct a consistent list of fully-instantiated `ACResource` objects and tell `ambassador.Config` to load these resources:

```python
aconf: Config = Config()
aconf.load_all(resources: List[ACResource])
```

`load_all` is only meant to be called once, with a complete set of all the resources comprising the Ambassador configuration. To change the configuration, instantiate a new `ambassador.Config`.

The `ambassador.Config` class is relatively unsophisticated: for the most part, all it does is to save the resources handed to it in a way that preserves the type of the resources, and permits consumers to query the `Config` for various kinds of resources:

* `get_config(self, key: str) -> Any`

    Fetches all the configuration information the `Config` has for the type tagged as `key`, e.g. `aconf.get_config("mappings")` will fetch all the `Mapping`s that the `Config` has stored.

    Current keys include: 
    
    - `auth_configs`: information from `AuthService` definitions
    - `mappings`: information from `Mapping`s
    - `modules`: information from `Module`s, including the `Ambassador` and `TLS` `Module`s
    - `ratelimit_configs`: information about `RateLimitService`s
    - `tracing_configs`: information about `TracingService`s

* `get_module(self, module_name: str) -> Optional[ACResource]`

    Fetches the `Module` with a given name (e.g. `aconf.get_module("ambassador")` will fetch information about the `Ambassador` `Module`). Returns `None` if no `Module` with the given name is found.

* `module_lookup(self, module_name: str, key: str, default: Any=None) -> Any` 

    Looks up a specific `key` in a specific `Module`. e.g.

    ```aconf.module_lookup('ambassador', 'service_port', 8080)```

    will look up the `service_port` from the `Ambassador` `Module`; if no `service_port` is defined, the default value will be 8080.

* `dump(self, output=sys.stdout) -> None`

    Dumps the entire `Config` object to the given `output`, for debugging.

Once a `Config` has loaded resources, it can be used to create an `IR`.

### The IR

The Intermediate Representation is where most of the logic of Ambassador lives. The IR as a whole is represented by an `ambassador.IR` object, which contains many objects descended from `ambassador.ir.IRResource`. The logic needed to synthesize the IR from an `ambassador.Config` is mostly contained in these `IRResource` subclasses; relatively little is in the `ambassador.IR` class itself.

An `ambassador.IR` can only be instantiated from an `ambassador.Config`:

```
ir = IR(aconf: Config)
```

The general rule in the `IR` is that everything interesting is stored in objects descended from `IRResource`.  There are two cases:

#### Only One `IRResource`

Only one of some `IRResource` objects can exist within a given IR, no matter how many distinct `ACResource`s they collect information from. A good example here is the `IRAmbassador` object, which contains global configuration information pooled from the `Ambassador` `Module` as well any `TLS` `Module`: only one of these will ever exist.

For these single-instance objects:

1. First the resource is instantiated with the `ambassador.IR` and `ambassador.Config` objects as parameters.   Whatever information is needed to initialize the `IRResource`, its `__init__` method should pull directly from the `Config` and/or `IR` objects.

2. Next, the resource's `setup` method is called, again with the `IR` and `Config` as parameters. `setup` can perform further initialization, consistency checks, etc., and must return a boolean: `True` will allow the resource to become active, `False` will mean the resource isn't needed after all. (For example, the `IRAuth` object will return `False` if it doesn't find any configured authentication services.)

    The default `setup` method simply returns `True`, so any `IRResource` subclass that doesn't override it will always be active.

3. If the resource's `setup` returns `True`, the `IR` will save it as an active resource.

4. After all resources have been probed, the `IR` will walk the list of active resources and call the `add_mappings` method for each active resource. `add_mappings` should add any mappings or clusters needed by the resource. (For example, the `IRAmbassador` object adds mappings (and thus clusters) to handle probes and diagnostics.)

    The default `add_mappings` method does nothing, so no mappings or clusters will be added for `IRResource` subclasses that don't override `add_mappings`.

For single-instance objects, the distinction between `__init__` and `setup` is rarely relevant. It's OK to do everything in `__init__` and allow the default `setup` to always return `True`; likewise it's OK to just save incoming data in `__init__` and have all logic in `setup`.

#### Multiple `IRResource`s

There can be multiple instances of some `IRResource` objects: for example, a single `Config` can contain many `Mapping`s and multiple `Listener`s. For these resources, we use a `Factory` class, which must implement the `load_all` classmethod and may implement the `finalize` classmethod, both of which receive the `IR` and `Config` as parameters.

1. `load_all` is called early in the instantiation process, and is responsible for creating as many individual resources as needed and saving them (usually in the `IR` itself).

2. `finalize` is called only after all the factories have had `load_all` called and all other single-instance resources have had `add_mappings` called, and is responsible for any normalization or other initialization that depends on global knowledge (for example, `MappingFactory.finalize` does the normalization of weights across `Mapping` groups).  

#### The Full Sequence

The full IR instantiation sequence can be found in `ambassador.ir.IR.__init__`:

- TLS defaults
- `IRAmbassadorTLS` (TLS contexts, etc.)
- `IRAmbassador` (global Ambassador config)
- `IRAuth` (extauth services)
- `IRRateLimit` (rate limiting)
- `ListenerFactory` (listeners -- creates `IRListener` objects)
- `MappingFactory` (mappings -- creates `IRMappingGroup`, `IRMapping`, and `IRCluster` objects)
- Cluster naming normalization

#### Helpers

- `IR.add_mapping` adds or looks up an `IRMappingGroup` with associated `IRMapping`s and `IRCluster`s.
- `IR.add_cluster` adds only an `IRCluster`.
- `IR.has_cluster` and `IR.get_cluster` do `IRCluster` lookups.
- `IR.dump` dumps the entire IR for debugging.

### The Envoy V1 Configuration

Finally, an Envoy V1 configuration is represented by `ambassador.envoy.V1Config`. A V1 config is built from an IR, again with most of the logic to do so contained in classes that mirror the structure of the V1 config. As we support later Envoy configuration versions, they will have their own classes. 

The root of the Envoy V1 configuration is `ambassador.envoy.v1.V1Config`.

Overall Life Cycle
------------------

0. Construct a collection of `ACResource` objects (from disk, from K8s, whatever).

    This will mostly involve `ACResource.from_dict` or `ACResource.from_yaml`.
    
1. Instantiate an `ambassador.Config`. Use its `load_all()` method to load up the collection of `ACResource` objects.

2. Instantiate an `ambassador.IR` from the `ambassador.Config`.

3. Instantiate an `ambassador.envoy.V1Config` from the `ambassador.IR`.  

Developing in Ambassador
------------------------

In all cases, understanding the class hierarchy and the lifecycle around the IR will be important. Both of these are discussed below. 

### Adding Features

Adding a feature will start with the Ambassador configuration resources:
 
- The simple case will involve modifying a schema file and possibly modifying an `ACResource` class.
- The less simple case will involve adding a new schema and a new `ACResource` subclass.
   - Unless the new class needs complex logic (it shouldn't), you can just let the existing `Config` code save your new resource.
   - If it does need complex logic, you'll need to add a handler method to `Config`.
   
Once the Ambassador config is dealt with, you'll add or modify the IR to cope. Most of what you do here should involve the `IRResource` subclasses, _not_ the `IR` class itself (although if you're adding something completely new, you'll need to add code to `IR.__init__` to call your new class).

Once the IR is dealt with, you'll need to add or modify the `V1Config` to cope with the IR changes. 

### Handling Bugs

The trick with bugs will be figuring out what you need to change. In general, work from V1Config to IR to Ambassador config -- the closer to Envoy that the fix can go, the simpler it will probably be.

- `ambassador.Config` and `ambassador.IR` both have `dump` methods that are invaluable for studying their contents.
- You can also attach debuggers and look at objects. Most of the things you're working with are subclasses of `dict`, so many introspection tools are simple.

Class Hierarchy
----------------

   * `IR`
      * `IRResource`
         * `IRAdmin`
         * `IRAmbassador`
         * `IRCluster`
         * `IRFilter`
            * `IRAuth`
         * `IRListener`
         * `IRMapping`
         * `IRMappingGroup`
         * `IRRateLimit`
         * `IRAmbassadorTLS`
         * `IREnvoyTLS`
   * `envoy`
      * `V1Config`
         * `V1Admin`
         * `V1Cluster`
         * `V1ClusterManager`
         * `V1Listener`

(This diagram is mostly about the way the classes are used, rather than strictly reflecting implementation. For example, the `Config` class is actually `ambassador.config.config.Config` but is imported into `ambassador` to make usage easier.)

The `Resource` Class
-------------------- 

`IRResource` and `ACResource` are subclasses of `ambassador.Resource`, although they are shown in the packages where they are logically used. A `Resource` is a kind of `dict` that can keep track of where it came from, what makes use of it, and any errors associated with it.

To initialize a `Resource` requires:

* `rkey`: a short identifier that is used as the primary key for _all_ the Ambassador classes to identify this single specific resource. It should be something like "ambassador-default.1" or the like that is very specific, though it doesn't have to be fun for humans.

* `location`: a more human-readable string describing where the human should go to find the source of this resource. "Service ambassador, namespace default, object 1". This is primarly used for diagnostics.

* `kind`: the kind of resource this is -- "Mapping", "TLS", "AuthService", whatever.

* `serialization`: the _original input serialization_, if we have it, of the object. If we don't have it, this should be `None` -- don't just serialize the object to no purpose.

All of these should be passed as keyword arguments (although it is possible to pass `rkey` and `kind` as positional arguments, it is discouraged). A `Resource` can also accept any other arbitrary keyword arguments, which will be saved in the `Resource` as they would be in a `dict`.

`Resource` defines multiple common methods and mechanisms:

* Dot notation and brace notation are equivalent for `Resource`: `rsrc.foo` and `rsrc["foo"]` are, by definition, equivalent.

* `rsrc.post_error(status: RichStatus)` posts an error notification that will be tracked throughout the life of the object. It will be the case that anything trying to use the `Resource` will inherit its errors; this is not fully implemented yet.

* `rsrc.referenced_by(other: Resource)` marks this `Resource` as being referenced by another `Resource`. For example, an Ambassador `Mapping` will cause Envoy clusters to be created. The `IRCluster` object created to track the cluster will reference the `ACMapping` that defined the mapping that caused the cluster to be created. This gives the diagnostics service a way to track from the cluster back to the annotation that caused it to exist.

* `rsrc.references(other: Resource)` is the other direction: it marks the other `Resource` as being referenced by us.

* `rsrc.is_referenced_by(rkey: str) -> Optional[Resource]` returns `None` if the given `rkey` is not associated with a `Resource` that references this `Resource`, or the referencing `Resource` if it is.

* `rsrc.as_dict() -> dict` returns a raw-dictionary form of just the data fields of the `Resource`. Things like the location and the references table are removed. 

* Class method `Resource.from_dict(rkey: str, location: str, serialization: Optional[str], attrs: dict)` creates a new `Resource` or subclass from a dictionary. The `kind` passed in the dictionary determines the actual class of the object returned (see below for more).

* Class method `Resource.from_yaml(rkey: str, location: str, serialization: str)` deserializes the YAML `serialization` and passes that to `Resource.from_dict`.

* Class method `Resource.from_resource(...)` clones a `Resource`, allowing optionally overriding any field using keyword arguments.

The `ACResource` class
----------------------

`ACResource` is a subclass of `Resource` which specifically refers to Ambassador configuration resources. It adds no new behaviors, but two additional keyword arguments are present when initializing:

* `name` is the name given to this Ambassador resource. Required for every type except `Pragma`.

* `apiVersion` is the API version to use when interpreting this resource. If not given, it defaults to "ambassador/v0".

Also, `ACResource.from_dict` will look first for `ACResource` subclasses when interpreting types.

The `IRResource` class
----------------------

`IRResource` is a subclass of `Resource` which specifically refers to IR resources. `IRResource` doesn't add any new fields (and, in fact, many `IRResource` subclasses default some fields) but several new behaviors are added.

