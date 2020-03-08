============
Introduction
============

This test infrastructure is designed to support testing of the wide
variety of steady-state behaviors for which ambassador can be
configured. It has the following requirements:

 - **speed** and **ease** of test creation (minimal boilerplate required)
 - **readability** (easy to understand what a test is doing even if you didn't write it)
 - **performance** (runs in a few seconds)
 - **scalability** (continues to run fast as we add more tests)
 - **composability**:

   + easy to express tests of combinatorial configuration inputs
   + tests of any given functionality can be expressed in a way that
     they can be reused wherever that functionality is valid, e.g.:

     - an option test can be used in isolation or combined with other option tests
     - any mapping test can be used with plain, http, auth, rate-limited
       ambassador configurations

Theory of Operation
-------------------

This testing infrastructure uses a customized execution, aggregation,
and parametrization model for tests. It is integrated with py.test
where they fit together well, but is different in ways that are
helpful to understand up front.

Execution Model
---------------

The default py.test execution model for the most part assumes each
test is relatively standalone and executing tests involves invoking
setup, run, and teardown for each test.

The execution model used by this test harness provides significantly
better performance and scalability for ambassador tests by defining
test execution in several phases designed to perform all synchronous
operations in batches. Test execution consists of the following
phases:

1. The manifests phase:

   - for each test gather kubernetes manifests from that test
   - apply all kubernetes manifests in aggregate *if* they have changed

2. The query phase:

   - for each test gather queries to make
   - pass all queries into a traffic driver that captures the results
     for later analysis

3. The check phase:

   - for each test perform any test-specific validation on the results
     of the queries supplied by that test

This execution model results in excellent performance and scalability
for tests that need to setup kubernetes manifests and perform
queries. For these sorts of tests, any number of them can be run with
just a single kubectl apply and a single batched traffic run. The
kubectl apply is skipped when the manifests don't change, and when
there are only small changes, it only performs a diff against the
cluster resulting in a super high speed common case for
development. The traffic driver is written similarly to a load testing
tool (batching/async with tunable concurrency) and as such can drive
test traffic through ambassador at close to peak envoy performance.

Aggregation Model
-----------------

The default py.test aggregation model organizes tests into flat
groups. These groups can depend on common infrastructure (e.g. a
database connection) by using fixtures.

The aggregation model used by this test harness instead organizes
tests into trees. Tests can be executed in the context of a parent
test, and tests can have child tests.

This both provides for execution efficiency, since tests can be
structured in such a way as to leverage the setup overhead of their
parent(s), but it is also helpful for expressing testing of different
ambassador input permutations, since tests can be written in such a
way as to be reusable in a variety of different structures, e.g. with
different parent tests and/or different child tests.

Parametrization Model
---------------------

The default py.test parametrization model involves repeatedly
executing the same test with different arguments.

The parametrization model used by this test harness extends this to
allow for tests to specify sets of other tests as parameters. This
compliments the aggregation model by providing a concise way to
express the different permutations of tests to execute.

Example
-------

As an example (which we will build up to), using this test
infrastructure it is possible to write an ambassador test that
executes with different top-level configurations, and contains any
mapping tests as children. It is also possible to write a mapping test
that exercises each mapping option both individually and in
combination with all the other options.

Integration with py.test
------------------------

The test harness integrates with py.test primarily in order to provide
a standard command line UX and to benefit from py.tests assertion and
reporting functionality.

Integration is accomplished by a `Runner` class that understands how
to discover and run tests written against this test harness, and
exposes each node in the resulting test tree(s) as individual py.test
test cases. Each py.test test case is named with a dot separated path
identifying the node in the test tree.

The py.test test cases are ordered as a pre-order traversal of the
test tree.

========
Tutorial
========

Gotchas/Prerequisites
---------------------

 - The tooling/build in this directory isn't particularly well
   integrated with the rest of the project, and isn't particularly
   mature. It's fairly simple, but you may need to use the Makefile
   contents as documentation for how to get stuff running.

 - The harness (and thus the tutorial) expects you to have access to a
   kubernetes cluster. An empty kubernaut should be fine, although it
   is very easy to express a lot of permutations, and so we may need
   bigger clusters at some point soon. (It would also be pretty
   straightforward to modify the harness to limit the batch size if
   needed.)

 - Because the harness works in batches, there is generally a pause up
   front and then all the tests run really fast. I recommend supplying
   the `-s` option to py.test since the harness will report progress
   to stdout and this pause will be less confusing.

 - The test suite will take much longer the first time you run it
   because none of the kubernetes resources have been created yet and
   so it needs to wait for them to spin up. Any subsequent runs should
   be much faster since the resources will either not need to be
   touched at all, or only patched slightly.

 - The test suite creates a bunch of /tmp/k8s-* files to store/compare
   yaml between runs. If you want to "clear the cache" you can remove
   these to get a clean run. This stuff could probably use pytest
   caching extensions instead.

Running tests
-------------

1. Get a kubernetes cluster and make sure your kubectl is pointed to it by default.

2. Run `py.test -s` from the appropriate directory:

 - cd ${BLAH}/poc
 - py.test -s

Note that the first time the tests run all the resources will need to
be created from scratch (as opposed to just patched), so they will
take a few minutes. Subsequent test runs should only take a few
seconds.

Also note, the readiness heuristics might not be suitably tuned for
all environments, so the queries may fail the first time they
run. Just try running the tests again if this happens.

Listing tests
-------------

If you want to see the test tree listed out, you can do the following:

1. Run `py.test --collect-only` from the appropriate directy:

 - cd ${BLAH}/poc
 - py.test --collect-only

You should see the full test tree flattened into a bunch of py.test
tests named according to their path within the tree.

Basic test skeleton
-------------------

An individual test defines methods that correspond to each phase of
execution. These are all optional, but provided here for illustration:

.. testsetup:: *

   import pytest, kat
   from typing import Optional, Sequence
   from kat.harness import abstract_test, sanitize, variants, Test, Query, Result, Runner

   kat.harness.DOCTEST = True

   def is_good(r): return True

.. doctest::

  >>> class ExampleTest(Test):
  ...
  ...     # perform test initialization, gets passed args from constructor
  ...     def init(self, *args, **kwargs) -> None:
  ...         pass
  ...
  ...     # return any kubernetes manifests needed for this test
  ...     def manifests(self) -> Optional[str]:
  ...         pass
  ...
  ...     # return any queries the probe should make
  ...     def queries(self) -> Sequence[Query]:
  ...         yield Query("https://www.google.com/") # expected defaults to 200
  ...         yield Query("https://www.google.com/blah", expected=404)
  ...
  ...     # filled with completed query results before check() is invoked
  ...     results: Sequence[Result]
  ...
  ...     # queries are checked automatically based on expected results,
  ...     # but this method allows additional checks
  ...     def check(self) -> None:
  ...         for r in self.results:
  ...             assert is_good(r)

We will step through each one of these methods in detail, but first we
need to be able to run our tests.

Running tests with py.test
--------------------------

Since py.test doesn't know how to run tests that look like this, we
need an adapter. The `Runner` class provides an adapter that will run
groups of these tests all at once. A runner is constructed with one or
more classes. The runner will discover all sub classes and run the
full set of tests as a group:

.. doctest::

  >>> t = Runner(ExampleTest)

The runner class defines some hooks that allow py.test to discover any
instances of this class automatically if you stick it in an
appropriately named file (i.e. any file starting with "test\_"). For
the rest of this tutorial, we can see the same thing by hand by
invoking `t.run()`:

.. doctest::

  >>> t.run()
  Querying 2 urls... done.
  ExampleTest: PASSED

Writing a Test with a Manifest
------------------------------

By defining a `manifests` method, we can deploy resources to
kubernetes as part of our test:

.. doctest::

  >>> class ManifestTest(Test):
  ...
  ...     def manifests(self):
  ...         return """
  ... ---
  ... kind: Service
  ... apiVersion: v1
  ... metadata:
  ...   name: hello-svc
  ... spec:
  ...   selector:
  ...     backend: hello-pod
  ...   ports:
  ...   - protocol: TCP
  ...     port: 80
  ...     targetPort: 8080
  ... ---
  ... apiVersion: v1
  ... kind: Pod
  ... metadata:
  ...   name: hello-pod
  ...   labels:
  ...     backend: hello-pod
  ... spec:
  ...   containers:
  ...   - name: backend
  ...     image: rschloming/backend:3
  ...     ports:
  ...     - containerPort: 8080
  ...     env:
  ...     - name: BACKEND
  ...       value: hello-pod
  ... """
  >>> Runner(ManifestTest).run()
  Manifests changed, applying.
  ManifestTest: PASSED

Using the `format` method to make tests more generic
----------------------------------------------------

Our manifest test works great in isolation, but if we were to use the
test more than once in a single group, we would have a problem. Each
instantiation of the test will end up producing the same manifests. To
solve this we can use the format method. This is a convenience method
with which uses the builtin python format language to format strings
with parameters accessible from the test instances. The test instance
is passed in as the `self` parameter. In other words, `test.format(s)`
is just convenience for `s.format(self=test)`. We can see it in use
here:

.. doctest::

  >>> class FormattedManifestTest(Test):
  ...
  ...     def manifests(self):
  ...         return self.format("""
  ... ---
  ... kind: Service
  ... apiVersion: v1
  ... metadata:
  ...   name: {self.path.k8s}
  ... spec:
  ...   selector:
  ...     backend: {self.path.k8s}
  ...   ports:
  ...   - protocol: TCP
  ...     port: 80
  ...     targetPort: 8080
  ... ---
  ... apiVersion: v1
  ... kind: Pod
  ... metadata:
  ...   name: {self.path.k8s}
  ...   labels:
  ...     backend: {self.path.k8s}
  ... spec:
  ...   containers:
  ...   - name: backend
  ...     image: rschloming/backend:3
  ...     ports:
  ...     - containerPort: 8080
  ...     env:
  ...     - name: BACKEND
  ...       value: {self.path.k8s}
  ... """)
  >>> Runner(FormattedManifestTest).run()
  Manifests changed, applying.
  FormattedManifestTest: PASSED

Note that test classes define both `name` and `path` fields that are
special subclasses of `str` that include a `k8s` property that returns
a version of the name that is sanitized for safe use as a kubernetes
name.

The `manifests` library
-----------------------

Just to eliminate even more boilerplate, the harness comes with a
`manifests` module that defines an `AMBASSADOR` template and a
`BACKEND` template:

.. doctest::

  >>> from kat import manifests
  >>> print(manifests.BACKEND)
  <BLANKLINE>
  ---
  kind: Service
  apiVersion: v1
  metadata:
    name: {self.path.k8s}
  spec:
    selector:
      backend: {self.path.k8s}
    ports:
    - name: http
      protocol: TCP
      port: 80
      targetPort: 8080
    - name: https
      protocol: TCP
      port: 443
      targetPort: 8443
  ---
  apiVersion: v1
  kind: Pod
  metadata:
    name: {self.path.k8s}
    labels:
      backend: {self.path.k8s}
  spec:
    containers:
    - name: backend
      image: {environ[KAT_SERVER_DOCKER_IMAGE]}
      ports:
      - containerPort: 8080
      env:
      - name: BACKEND
        value: {self.path.k8s}
  <BLANKLINE>

For both efficiency and convenience, these templates define `pods`
directly rather than using `deployments` to create pods. This saves
some setup time/overhead, and is also much more convenient for
debugging since each pod ends up being directly named for the
(sanitized) test case that instantiates it rather than having the name
mangling introduced by an extra level of replica-set and deployment
objects surrounding the pod.

We can now define our manifest test much more concisely:

.. doctest::

  >>> class ConciseManifestTest(Test):
  ...
  ...     def manifests(self):
  ...         return self.format(manifests.BACKEND)
  ...
  >>> Runner(ConciseManifestTest).run()
  Manifests changed, applying.
  ConciseManifestTest: PASSED

There is one caveat with how we have used manifests so far. We need to
wait until resources are ready before actually continuing our
tests. To learn more about how this works go to the Combining
Manifests and Queries section, but first lets talk about making
queries.

Writing tests that perform Queries
----------------------------------

To write a test that performs a query, we define a `queries` generator
function that yields any number of `Query` objects. We can then access
the result of all those queries in the `check` method in exactly the
same order we yielded them. Queries are automatically checked for an
expected result. The default expected result is 200, if you want to
override this you can use the `expected` keyword argument:

.. doctest::

  >>> class QueryTest(Test):
  ...
  ...     def queries(self):
  ...         for i in range(10):
  ...             yield Query("http://httpbin.org/get?count=%s" % i)
  ...         yield Query("http://httpbin.org/status/404", expected=404)
  ...
  ...     def check(self):
  ...         for i, r in enumerate(self.results[:10]):
  ...             args = r.json["args"]
  ...             assert int(args["count"]) == i, args
  ...
  >>> Runner(QueryTest).run()
  Querying 11 urls... done.
  QueryTest: PASSED

Combining Manifests and Queries (using requirements)
----------------------------------------------------

Combining manifests and queries is almost as easy as just defining the
two methods with one catch. We need to tell the test harness how to
figure out when the resources defined in the manifests are ready to be
queried. To do this we define the `requirements` method to yield the
kind and name of resources that need to be ready. Let's use this to
run httpbin in our own cluster:

.. doctest::

  >>> class CombinedTest(Test):
  ...
  ...     def manifests(self):
  ...         return self.format(manifests.HTTPBIN)
  ...
  ...     def requirements(self):
  ...         yield ("pod", self.path.k8s)
  ...
  ...     def queries(self):
  ...         yield Query("http://%s/get?foo=bar" % self.path.k8s)
  ...
  ...     def check(self):
  ...         assert self.results[0].json["args"]["foo"] == "bar"
  ...
  >>> Runner(CombinedTest).run()
  Manifests changed, applying.
  Checking requirements... satisfied.
  Querying 1 urls... done.
  CombinedTest: PASSED

Writing tests with Ambassador configuration
-------------------------------------------

The test harness has some special convenience hooks for injecting
ambassador configuration into manifests. If you define a `config`
method, it can yield a tuple of a target test node and an ambassador
configuration input. The harness will automatically inject the
supplied ambassador yaml as an appropriate annotation on the manifests
associated with the target node:

.. doctest::

  >>> class Ambassador(Test):
  ...
  ...     def manifests(self):
  ...         return self.format(manifests.AMBASSADOR, image="quay.io/datawire/ambassador:0.35.3")
  ...
  ...     def requirements(self):
  ...        yield ("pod", self.name.k8s)
  ...
  ...     def config(self):
  ...         yield self, """
  ... ---
  ... apiVersion: ambassador/v2
  ... kind:  Module
  ... name:  ambassador
  ... config: {}
  ... """
  >>> Runner(Ambassador).run()
  Manifests changed, applying.
  Checking requirements... satisfied.
  Ambassador: PASSED

This isn't super interesting all by itself, but it gets more
interesting when composing tests, but first lets take a look at
parameterizing tests.

Parameterizing Tests
--------------------

If you want to instantiate a test multiple times, you can use the
`variants` classmethod to control how tests are instantiated. The
`variants` class method can yield as many variants as it likes of a
given class:

.. doctest::

  >>> class ParametrizedQuery(Test):
  ...
  ...     @classmethod
  ...     def variants(cls):
  ...         for url in ("http://httpbin.org", "http://google.com"):
  ...           yield cls(url, name=sanitize(url))
  ...
  ...     def init(self, url):
  ...         self.url = url
  ...
  ...     def queries(self):
  ...         yield Query(self.url)
  ...
  >>> Runner(ParametrizedQuery).run()
  Querying 2 urls... done.
  ParametrizedQuery-http-SCHEME-httpbin-DOT-org: PASSED
  ParametrizedQuery-http-SCHEME-google-DOT-com: PASSED


Composing Tests
---------------

In addition to using the `variants` classmethod to parameterize tests,
you can use it to compose tests. The `variants` *function* will return
all the variants of a given test case. You can use this to embed them
within another test, e.g.:

.. doctest::

  >>> class Composite(Test):
  ...
  ...     @classmethod
  ...     def variants(cls):
  ...         yield cls(variants(Mapping))
  ...
  ...     def manifests(self):
  ...         return self.format(manifests.AMBASSADOR, image="quay.io/datawire/ambassador:0.35.3")
  ...
  ...     def requirements(self):
  ...        yield ("pod", self.name.k8s)
  ...
  ...     def config(self):
  ...         yield self, """
  ... ---
  ... apiVersion: ambassador/v2
  ... kind:  Module
  ... name:  ambassador
  ... config: {}
  ... """

Note the use of the `variants` function to embed `Mapping` tests
within our `Composite` test. Now we can write our mapping test like
so:

.. doctest::

  >>> class Mapping(Test):
  ...
  ...     def manifests(self):
  ...         return self.format(manifests.HTTPBIN)
  ...
  ...     def requirements(self):
  ...         yield ("pod", self.path.k8s)
  ...
  ...     def config(self):
  ...         yield self, self.format("""
  ... ---
  ... apiVersion: ambassador/2
  ... kind:  Mapping
  ... name:  {self.name}
  ... prefix: /{self.name}/
  ... service: http://{self.path.k8s}
  ... """)
  ...
  ...     def queries(self):
  ...         yield Query("http://%s/%s/" % (self.parent.name.k8s, self.name))
  ...
  >>> Runner(Composite).run()
  Manifests changed, applying.
  Checking requirements... satisfied.
  Querying 1 urls... done.
  Composite: PASSED
  Mapping: PASSED

Note the use of the `parent` attribute to make the test portable. All
tests automatically have `parent`, `children`, `name`, and `path`
attributes supplied automatically.

Backend service features aka the Result class
---------------------------------------------

The backend service implementation provides a number of handy
features. It supports op-codes via headers that let the requestor
control the return status and let the requestor ask for specific
headers to be returned.

The backend service implementation logs everything about the incoming
request and outgoing response into a json structure that it returns in
the body. This is parsed by the `Result` class allowing tests to
access the request and response as seen/produced by the backend
service.

Using a base test for discovery
-------------------------------

By defining a base class we can avoid constructing lots of runners:

.. doctest::

  >>> @abstract_test
  ... class TutorialTest(Test):
  ...     pass
  ...
  >>> class TestA(TutorialTest):
  ...    pass
  ...
  >>> class TestB(TutorialTest):
  ...    pass
  ...
  >>> Runner(TutorialTest).run()
  TestA: PASSED
  TestB: PASSED


The `@abstract_test` annotation tells the Runner not to bother
instantiate that class directly as a test, however it will still
discover any subclasses.

Abstract Tests
--------------

The `abstract_tests` module defines a number of abstract test cases
using the techinques described above. Subclasses of `AmbassadorTest`
can define different core configuration options and will automatically
include all subclasses of `MappingTest`.

 - `AmbassadorTest`
 - `MappingTest`
 - `OptionTest`
