Ambassador Unit Tests
=====================

Ambassador is tested with [KAT](../../kat/README.md). **You are strongly encouraged to add tests when you add features.** However, KAT is fairly complex, and how you actually add will depend a bit on what kind of thing you're adding. Here we'll talk a bit about how Ambassador uses KAT, and how to work within that

The Ambassador KAT class hierarchy looks a bit like this:

```
Node
  ServiceType (abstract_tests.py)
    HTTP -- generic HTTP echo backend (abstract_tests.py)
    AHTTP -- authentication service using HTTP (abstract_tests.py)
    etc.
  Test
    AmbassadorTest (abstract_tests.py)
      TCP (test_ambassador.py)
      Plain (test_ambassador.py)
      TracingTest (t_tracing.py)
      etc.
    MappingTest (abstract_tests.py)
      SimpleMapping (test_ambassador.py)
    OptionTest (abstract_tests.py)
      AddRequestHeaders (test_ambassador.py)
      AddResponseHeaders (test_ambassador.py)
      etc.
```

Broadly speaking:

- an `AmbassadorTest` is a test that instantiates its own Ambassador, and can therefore safely modify things like the Ambassador `Module` and such;
- a `MappingTest` defines a `Mapping` that will be added to both the `Plain` and `TCP` `AmbassadorTests`;
- an `OptionTest` defines an option that will be added to all `MappingTests`; and
- a `ServiceType` is a kind of backend service you can talk to.

Historically, we started with all the tests lumped into `test_ambassador.py`. As this has become more and more unwieldy, we've split out into the `t_*.py` files. If you want to create a new file for your test, great! but don't give it a name starting with `test_` because that can confuse things.

All the way at the bottom of `test_ambassador.py` you'll find

```
main = Runner(AmbassadorTest)
```

which is what actually uses KAT to instantiate and run all subclasses of `AmbassadorTest` as tests.

### `OptionTest`

If you're lucky, your test can be an `OptionTest`: this is appropriate if you're adding an option to the `Mapping` class. Just check out an existing test (like `AddRequestHeaders`) and you'll be good to go.

Within an `OptionTest`:
- the `config` method should just `yield` the string you want to be added to the `Mapping` configuration for your option
- the `queries` method only needs to `yield` queries that the parent `MappingTest` won't already be doing. So, for example, the `AddRequestHeader` test doesn't have a `queries` method -- it will have an effect on all the queries that the parent is already performing.
- the `check` method can assert about `self.parent.results` to check things about queries made by the parent, or about `self.results` if the `queries` method added more queries.

### `MappingTest`

A `MappingTest` adds an entirely new `Mapping` to an `AmbassadorTest`'s Ambassador. `MappingTest`s are a bit more challenging to work with than `OptionTest`s.

- To initialize a `MappingTest`, you must pass it a `target`, which must be an instance of (a subclass of) `ServiceType` at minimum, and you may also pass it a tuple of `OptionTest`s (and of course a `name`). The minimal instantiation is something like

    `MappingTest(HTTP())`

  to create a `MappingTest` with our generic HTTP echo service for its `target`. The `target` is accessible in the `MappingTest` as `self.target`.

- The `variants` class method must `yield` valid instances of the `MappingTest`. The `WebSocketMapping` test is a good example:

    ```
    class WebSocketMapping(MappingTest):
        ...

        @classmethod
        def variants(cls):
            for st in variants(ServiceType):
                yield cls(st, name="{self.target.name}")
    ```

  Here we define `WebSocketMapping` as a subclass of `MappingTest`. It varies with `ServiceType`s -- for every kind of `ServiceType` out there, we take the instantiated `ServiceType` and instantiate a new `WebSocketMapping` with the instantiated `ServiceType` as the `target`. We need to change the `name` of our new instance, too, so that we don't have duplicates.

- `queries` and `check` should generate and check a new set of queries. Again, it's OK to look at `self.parent.results` in `check`.

### `AmbassadorTest`

An `AmbassadorTest` is the place that new Ambassadors actually get created. If your test doesn't need a new Ambassador, don't create a new `AmbassadorTest`! Each of these requires a new pod in the test cluster; they're somewhat expensive. On the other hand, you _need_ an `AmbassadorTest` if you need to play with the `Ambassador` `Module`, or TLS contexts, or things like that.

- By default, an `AmbassadorTest` uses an `ambassador_id` of its name. You can override this by setting `ambassador_id` on your instance, but why bother?

- By default, an `AmbassadorTest` will appear in namespace `default`. You can override this by setting `namespace` on your instance.

- If you set `single_namespace` on your instance, that will restrict your Ambassador to only looking in its namespace.

- The `config` method will likely need to `yield` tuples: `yield self, configstr` to set configuration on the Ambassador itself, or `yield self.target, configstr` to set configuration on a service you've saved as `self.target`.

- If you need to change `manifests`, you will almost certainly need to include a call to `super().manifests()` too. The common pattern here is

    def manifests(self):
        return manifeststr + super().manifests()

  so that you can be that the manifest in `manifeststr` takes effect before the Ambassador is created.

- A fairly common pattern for a single-purpose `AmbassadorTest` is to create a target as `self.target` in the `init` method (not `__init__` -- leave that alone!).

### More coming soon!
