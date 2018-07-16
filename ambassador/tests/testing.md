The Problem
-----------

We need to transition ambassador to using v2. This will involve:

1. write new code to translate from IR -> v2

2. run e2e tests

3. manually compare the v2 config produced by the new code against the
   v1 gold.json

Our confidence at the end of this process is based largely on our e2e
test coverage and the manual inspection step. We would like to quickly
improve this confidence by adding the right kind of tests.

#### What are the right kind of tests?

We are trying to establish confidence that any *likely* permutation of
ambassador input configuration will produce the desired behavior. This
includes invalid inputs as well as valid.

It's worth noting that code coverage metrics don't mean a whole lot
for us since a relatively artificial set of sample input
configurations can fairly easily provide 100% code coverage without
actually giving us confidence that realistic input permutations will
work properly.

What we really care about is not code coverage, but *input*
coverage. Figuring out of all the many different possible permutations
of input that can be supplied, which subset of those will give us the
most confidence in the face of real-world inputs. In order to figure
this out, we need to start not with the code, but with the input
schema.

There are roughly 4 different "dimensions" of input config:

1. global config (Module.schema)
2. auth config (AuthService.schema)
3. rate limiting (RateLimitService.schema)
4. mappings (Mapping.schema)

The mappings one is the most complex as there are roughly 32
properties at this point, resulting in many different possible
permutations that people might exercise.

We could brute force this problem by writing code to generate sample
inputs for each dimension and test every possible permuation, but this
would result in a combinatorial explosion. What we want to do is
identify a reasonable subset of this that will give us confidence and
figure out how to quickly/succinctly express tests for all the
reasonable combinations.

#### What kind of tests do we have now?

Our current e2e tests cover a select set of scenarios, e.g. canary
rollout. Tests of new features, e.g. additional mapping properties
have been added in an ad-hoc manner to these scenarios as is
convenient.

This combines testing of both the steady state behavior of a given
configuration with testing of dynamic behaviors when configuration is
changed.

Envoy is largely responsible for doing the right thing when
configuration changes, so we can largely ignore this problem for
ambassador testing and focus on two things:

1. Are we correctly relaying configuration updates to envoy.

2. Does a given steady-state configuration produce the desired
   behavior.

I'm going to assume we will tackle how to test (1) as part of the ADS
work, and so the rest of this really focuses on (2).

### Additional Problems

#### How to concisely specify combinatorial configuration inputs

A good example of this is the mapping schema which has a lot of
options that are supposed to work for a variety of different mapping
definitions and/or global configurations. For example,
add_request_headers, auto_host_rewrite, case_sensitive, timeout_ms,
weight, rate_limits, are all options that can/should do something
reasonable whether the service in question is grpc, http, https, and
if/when tcp happens then some of them may apply to tcp, and those that
don't should produce useful error messages.

This basic fact surfaces as a number of other problems, e.g.:

 - How do we test grpc?

   The current answer is just do it very minimally, but unless we have
   a way to cover all the combinations, one of grpc/https/tcp will be
   a second class citizen, or we will just have a patchwork of
   coverage.

 - How do I figure out where to put my tests?

   With the existing scenario based testing, if I'm adding a new
   mapping property, it's not at all clear where I would put a
   test. Another way to say this is that where there are clear
   categories of functionality, e.g. "mapping option", we need a clear
   category of tests to add to as we expand the functionality.

#### Depending on gold files as tests

It's also worth noting that we can't continue to depend on comparing
the output of the translator subset of ambassador against a known gold
file as a test. This technique is great if the format and semantics of
the output of the translator is fixed, but fundamentally doesn't work
against a moving target, and envoy has proven to be a moving target
for both format *and* semantics.

We *can* use the comparison technique purely as an opaque hash to
avoid unnecessary re-testing. I feel pretty strongly that this needs
to be transparent to the dev process, e.g. no ambassador dev should
ever need to look at the contents of these files or ever compare them
by hand, or even know they exist as anything other than an opaque
short-circuit mechanism unless they are hacking on the test harness
itself. I'm also tempted to say these files should never appear in
git, only as build/test artifacts. The reason being that storing them
in git lets us forget how expensive it is to recompute them until we
need to change envoy versions or we make a disruptive enough change to
the translator, at which point we could find out at a bad time about a
bunch of tech tebt we accumulated without realizing it.


The Solution (or at least a starting point)
-------------------------------------------

The basic idea is to change the granularity of what a "test" is. This
is very much started by differentiating between steady-state and
dynamic tests. By making this distinction we can move from a world
where a single test is a complete scenario, something like this:

 - setup config state 1
 - wait for config to take effect
 - drive traffic/do assertions
 - change to config state 2
 - wait for config to take effect
 - drive traffic/do assertions
 - ...

To a world where a single test is basically just a single
configuration along with test traffic and assertions to be applied
when that configuration reaches steady state:

 - setup config state
 - wait for config to take effect
 - drive traffic/do assertions

While this is way better, we are still left with the combinatorial
problem described above. So to address the combinatorial problem, we
can redefine the granularity of a test even more finely.

Instead of a test being a complete configuration, we instead define a
test based on a *mergable* fragment of configuration (as well as a
mergable traffic driver and assertion set) and make the test harness
be responsible for assembling the complete configuration, doing any
setup necessary, driving the combined set of traffic and doing the
combined set of assertions.

For configuration, this could work something like this:

1. We define a set of configuration fragment test types, e.g.:

   - GlobalConfig: (not sure what else would go here)
     + tls
     + plain

   - MappingDefinition:
     + grpc
     + http
     + ...

   - MappingOption:
     + ...

   Or possible something more like this:

   - GlobalConfig:
     + tls
     + plain

   - ServiceType:
     + grpc
     + http
     + ...

   - MappingDefinition:
     + fill out with different interesting matching permutations

   - MappingOption:
     + add_request_headers
     + case_sensitive
     + ...

2. Then our test harness looks something like this:


   ```
   permutations = []
   for gc in globalConfigs:
     extend_with(permutations, gc)
     for st in serviceTypes:
       for md in mappingDefinitions:
         extend_with(permutations, md, st)
         for mo in mappingOptions:
           extend_with(permutations, mo)
   ```

   Where extend_with takes the given config fragment/test and extends
   the existing set of permutations. (It might not actually be the
   same extend_with at each level of the loop, but you get the idea.)

I'm handwaving a bit, but hopefully it's easy to see that we can
generate a lot of permutations of valid as well as potentially invalid
configurations. This is all good, but we need to also be able to drive
traffic/apply assertions for each of these configurations.

#### How do we drive traffic in a mergable way?

The combination of GlobalConfig, ServiceType, and MappingDefinition
establish a basic set of urls and responses you can expect. So long as
the driver code in a MappingDefinition or MappingOption test has
access to the context into which it is merged, then the test could
produce a complete url(s) to query, e.g.:

```
  # test case_sensitive = false
  def drive_traffic(self, ctx):
    base_url = "%s://%s/%s" % (ctx.protocol, ctx.host, ctx.prefix)
    upper_url = "%s://%s/%s" % (ctx.protocol, ctx.host, ctx.prefix.upper)
    lower_url = "%s://%s/%s" % (ctx.protocol, ctx.host, ctx.prefix.lower)
    ...
```

#### How do we specify assertions in a mergable way?

For every url we test we can collect the following:

1. The request and response as sent to/seen from envoy.
2. The request and response as sent to/seen from any backend services
   it hits.

Doing this implies:

1. That we inject a unique request-id into each test request.
2. That every backend test service (grpc, http, etc) logs incoming
   requests and outgoing responses along with the request-id.

This information should be hopefully enough to perform most of the
assertions we care about, e.g.:

```
  # test add_request_headers
  def drive_traffic(self, ctx):
    base_url = ctx.url
    info = self.request(url)
    assert "added" in info.backend_requests[ctx.target].headers

  # test shadow
  def drive_traffic(self, ctx):
    base_url = ctx.url
    info = self.request(url)
    assert self.shadow_target in info.backend_requests
```

We probably need a bit more, e.g. any diagnostic errors would be
needed for negative testing.

#### How does this all come together?

The test driver would look something like this:

1. Collect all the specified tests.

2. Generate all the test permutations from the collected tests.

3. Query the test permutations for all the backend services needed for
   testing.

4. Spinup backend services (this could be done locally in docker or in
   kubernetes).

5. Configure ambassador(s)

6. Run all the test permutations in order to drive traffic, gather
   request/response data, and perform assertions.

In dev mode for steady state testing we could probably just leave most
of the setup in place (possibly making incremental changes). This
would cut down the fast path in the dev case to something like:

1. Step 5 (for the subset of tests you care about)
2. Step 6 (for the subset of tests you care about)
