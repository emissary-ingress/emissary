Developing Ambassador
=====================

<!--
  When editing this document, the heirarchy of headings is:

     Heading 1
     =========

     Heading 2
     ---------

     ### Heading 3

     #### Heading 4
-->

Ambassador is a complex piece of software with lots of integrations
and moving parts. Just being able to build the code and run tests is
often not sufficient to work efficiently on a given piece of the
code. This document functions as a central registry for how to
**efficiently** hack on any part of ambassador.

How do I get ambassador without building it?
--------------------------------------------

Check out https://www.getambassador.io/!

How do I get help with any of this stuff?
-----------------------------------------

Ask on our [Slack channel](https://d6e.co/slack) in the `#ambassador-dev` channel.

How do I setup a system for ambassador development?
---------------------------------------------------

To build or hack on ambassador, there are a number of
prerequisites. In general our tooling tries to detect any missing
requirements and provide a friendly error message. If you ever find
that this is not the case please file a PR with a fix. Likewise if you
ever find anything missing from this list.

### Requirements:

 - git
 - make
 - docker (make sure you can run docker commands as your dev user without sudo)
 - bash
 - rsync (with the --info option)
 - golang 1.13
 - python 3.7+
 - kubectl
 - a kubernetes cluster
 - a docker registry

### Configuration:

 - `export DEV_REGISTRY=<your-dev-docker-registry>` (you need to be logged in and have permission to push)
 - `export DEV_KUBECONFIG=<your-dev-kubeconfig>` (your cluster needs to be able to read from your registry,
                                                  specifically from the ambassador, kat-server, and kat-client repos)
 - `export GCLOUD_CONFIG=<your-config>` (only needed if your kubeconfig uses gcloud, which is likely for a GKE cluster)

Please note that ambassador tests and build system will do destructive
things to your development cluster. We therefore recommend that you
create a separate kubeconfig file dedicated for ambassador development
and point DEV_KUBECONFIG to this file instead of using the default
`~/.kube/config` location.

How do I find out what build targets are available?
---------------------------------------------------

Use `make help` and `make targets` to see what build targets are
available along with documentation for what each target does.

How do I build an ambassador image from source?
-----------------------------------------------

0. `git clone https://github.com/datawire/ambassador.git && cd ambassador`
1. `make images` (this will take a while the first time)

The ambassador image will be tagged as `ambassador:latest`. There will
also be a `kat-server:latest` and a `kat-client:latest` image. These
two images are only used for testing.

How do I push an ambassador image from source?
----------------------------------------------

1. `export DEV_REGISTRY=<your-dev-docker-registry>` (you need to be logged in and have permission to push)
2. `make push`
3. The output will contain the image names. You can also display this using
   `make env` or `make export`. The latter form is suitable for
   passing to bash.

NOTE: This will also push the `kat-client` and `kat-server` images.

How do I deploy an ambassador to a cluster from source?
-------------------------------------------------------

XXX: This does not work yet, but will be fixed in a future commit!!!

1. `export DEV_REGISTRY=<your-dev-docker-registry>` (you need to be logged in and have permission to push)
2. `export DEV_KUBECONFIG=<your-dev-kubeconfig>`
3. `make deploy`

How do I run ambassador tests?
------------------------------

- `export DEV_REGISTRY=<your-dev-docker-registry>` (you need to be logged in and have permission to push)
- `export DEV_KUBECONFIG=<your-dev-kubeconfig>`

| Group           | Command                                                             |
|-----------------|---------------------------------------------------------------------|
| All Tests       | `make test`                                                         |
| All Golang      | `make gotest`                                                       |
| All Python      | `make pytest`                                                       |
| Some/One Golang | `make gotest GOTEST_PKGS=./cmd/edgectl GOTEST_ARGS="-run TestName"` |
| Some/One Python | `make pytest PYTEST_ARGS="-k TestName"`                             |

Please note the python tests use a local cache to speed up test
results. If you make a code update that changes the generated envoy
configuration, those tests will fail and you will need to update the
python test cache.

How do I update the python test cache?
--------------------------------------

- First, run `make KAT_RUN_MODE=envoy pytest` to do a test run _without_
  using the local cache.

- Once that succeeds, use `make pytest-gold` to update the cache from
  the passing tests.

How do I debug/develop envoy config generation?
-----------------------------------------------

Envoy configuration is generated by the ambassador compiler. Debugging
the ambassador compiler by running it in kubernetes is very slow since
we need to push both the code and any relevant kubernetes resources
into the cluster.

### `mockery`

Fortunately we have the `mockery` tool which lets us run the compiler
code directly on kubernetes resources without having to push that code
or the relevant kubernetes resources into the cluster. This is the
fastest way to hack on and debug the compiler.

The `mockery` tool runs inside the Docker container used to build
Ambassador, using `make shell`, so it's important to realize that it
won't have access to your entire filesystem. There are two easy ways
to arrange to get data in and out of the container:

1. If you `make sync`, everything in the Ambassador source tree gets rsync'd
   into the container's `/buildroot/ambassador`. The first time you start the
   shell, this can take a bit, but after that it's pretty fast. You'll
   probably need to use `docker cp` to get data out of the container, though.

2. You may be able to use Docker volume mounts by exporting `BUILDER_MOUNTS`
   with the appropriate `-v` switches before running `make shell` -- e.g.

    ```
    export BUILDER_MOUNTS=$(pwd)/xfer:/xfer
    make shell
    ```

   will cause the dev shell to mount `xfer` in your current directory as `/xfer`.
   This is known to work well on MacOS (though volume mounts are slow on Mac,
   so moving gigabytes of data around this way isn't ideal).

Once you've sorted out how to move data around:

1. Put together a set of Ambassador configuration CRDs in a file that's somewhere
   that you'll be able to get them into the builder container. The easy way to do
   this is to use the files you'd feed to `kubectl apply`; they should be actual
   Kubernetes objects with `metadata` and `spec` sections, etc. (If you want to
   use annotations, that's OK too, just put the whole `Service` object in there.)

2. Run `make compile shell` to build everything and start the dev shell.

3. From inside the build shell, run

   ```
   mockery $path_to_your_file
   ```

   If you're using a non-default `ambassador_id` you need to provide it in the
   environment:

   ```
   AMBASSADOR_ID=whatever mockery $path_to_your_file
   ```

   Finally, if you're trying to mimic `KAT`, copy the `/tmp/k8s-AmbassadorTest.yaml`
   file from a KAT run to use as input, then

   ```
   mockery --kat $kat_test_name $path_to_k8s_AmbassadorTest.yaml
   ```

   where `$kat_test_name` is the class name of a `KAT` test class, like `LuaTest` or
   `TLSContextTest`.


4. Once it's done, `/tmp/ambassador/snapshots` will have all the output from the
   compiler phase of Ambassador.

The point of `mockery` is that it mimics the configuration cycle of real Ambassador,
without relying at all on a Kubernetes cluster. This means that you can easily and
quickly take a Kubernetes input and look at the generated Envoy configuration without
any other infrastructure.

### `ambassador dump`

The `ambassador dump` tool is also useful for debugging and hacking on
the compiler. After running `make shell`, you'll also be able to use
the `ambassador` CLI, which can export the most import data structures
that Ambassador works with as JSON.  It works from an input which can
be either a single file or a directory full of files in the following
formats:

- raw Ambassador resources like you'll find in the `demo/config` directory; or
- an annotated Kubernetes resources like you'll find in `/tmp/k8s-AmbassadorTest.yaml` after running `make test`; or
- a `watt` snapshot like you'll find in the `$AMBASSADOR_CONFIG_BASE_DIR/snapshots/snapshot.yaml` (which is a JSON file, I know, it's misnamed).

Given an input source, running

```
ambassador dump --ir --v2 [$input_flags] $input > test.json
```

will dump the Ambassador IR and v2 Envoy configuration into `test.json`. Here
`$input_flags` will be

- nothing for raw Ambassador resources;
- `--k8s` for Kubernetes resources; or
- `--watt` for a `watt` snapshot.

You can get more information with

```
ambassador dump --help
```

How do I type check my python code?
-----------------------------------

XXX: the `make mypy` target does not exist yet, a future commit will fix this!

Ambassador uses Python 3 type hinting and the `mypy` static type checker to
help find bugs before runtime. If you haven't worked with hinting before, a
good place to start is
[the `mypy` cheat sheet](https://mypy.readthedocs.io/en/latest/cheat_sheet_py3.html).

New code must be hinted, and the build process will verify that the type
check passes when you `make test`. Fair warning: this means that
PRs will not pass CI if the type checker fails.

We strongly recommend using an editor that can do realtime type checking
(at Datawire we tend to use PyCharm and VSCode a lot, but many many editors
can do this now) and also running the type checker by hand before submitting
anything:

- `make mypy` will start check all the Ambassador code

Since `make mypy` uses the daemon for caching, it should be very fast after
the first run. Ambassador code should produce _no_ warnings and _no_ errors.

If you're concerned that the cache is somehow wrong (or if you just want the
daemon to not be there any more), `make mypy-clean` will stop the daemon
and clear the cache.

How do I make documentation-only changes?
-----------------------------------------

The Ambassador documentation lives in the `docs` directory. If you're working
on documentation for an upcoming feature or fix, make your docs changes along
with your code changes, and include them all in the same PR.

If you want to make a change that **only** affects the live documentation for
an already-released version of Ambassador, you'll need to make your changes in
a branch from the `release` branch for that version, then PR back to the
`release` branch. For example, if you find a typo while reading the documentation
for Ambassador 1.4:

- Check out `release/v1.4`
- Make a branch from it.
- Fix the typo.
- Push your branch and PR it back to `release/v1.4`.

How do I get the source code for a release?
-------------------------------------------

The current shipping release of Ambassador lives on the `master`
branch. It is tagged with its version (e.g. `v0.78.0`).

Changes on `master` after the last tag have not been released yet, but
will be included in the next release of Ambassador.

How do I make a contribution?
-----------------------------

1. **All development must be on branches cut from `master`**.
   - We recommend that your branches start with your username.
      - At Datawire we typically use `git-flow`-style naming, e.g. `flynn/dev/telepathic-ui`
   - Please do not use a branch name starting with `release`.

2. If your development takes any significant time, **merge master back into your branch regularly**.
   - Think "every morning" and "right before submitting a pull request."
   - If you're using a branch name that starts with your username, `git rebase` is also OK and no one from Datawire will scream at you for force-pushing.
   - Please do **not** rebase any branch that does **not** start with your username.

3. **Code changes must include relevant documentation updates.**
   - Make changes in the `docs` directory as necessary, and commit them to your
     branch so that they can be incorporated when the feature is merged into `master`.

4. **Code changes must include passing tests.**
   - See `python/tests/README.md` for more here.
   - Your tests **must** actually test the change you're making.
   - Your tests **must** pass in order for your change to be accepted.

5. When you have things working and tested, **submit a pull request back to `master`**.
   - Make sure your branch is up-to-date with `master` right before submitting the PR!
   - The PR will trigger CI to perform a build and run tests.
   - CI tests **must** be passing for the PR to be merged.

6. When all is well, maintainers will merge the PR into `master`, accepting your
   change for the next Ambassador release. Thanks!

How do I make changes to the Envoy that ships with Ambassador?
--------------------------------------------------------------

This is a bit more complex than anyone likes, but here goes:

### 1. Preparing your machine

Building and testing Envoy can be very resource intensive.  A laptop
often can build Envoy... if you plug in an external hard drive, point
a fan at it, and leave it running overnight and most of the next day.
At Datawire, we'll often spin up a temporary build machine in GCE, so
that we can build it very quickly.

As of Envoy 1.14.4, we've measure the resource use to build and test
it as:

> | Command            | Disk Size | Disk Used | Duration[1] |
> |--------------------|-----------|-----------|-------------|
> | `make update-base` | 400G      | 12GB      | ~11m        |
> | `make check-envoy` | 400G      | 339GB     | ~45m        |
>
> [1] On a "Machine type: custom (32 vCPUs, 512 GB memory)" VM on GCE,
> with the following entry in its `/etc/fstab`:
>
> ```
> tmpfs:docker  /var/lib/docker tmpfs size=400G 0 0
> ```

In the past, we've seen tests fail or refuse to run if there's "low"
diskspace, even though there's still a lot left; for instance, with
Envoy 1.13, even though the total disk used by a passing test run is
only 211GB, about half of the tests would fail or refuse to run on a
250GB drive.  Since then, we've been dogmatically using a 400GB drive,
which is still enough as of Envoy 1.14.4.

If you have the RAM, we've seen huge speed gains from doing the builds
and tests on a RAM disk (see the `/etc/fstab` line above).

### 2. Setting up your workspace to hack on Envoy

1. From your `ambassador.git` checkout, get Ambassador's current
   version of the Envoy sources, and create a branch from that:

   ```shell
   make $(pwd)/cxx/envoy
   git -C cxx/envoy checkout -b YOUR_BRANCHNAME
   ```

2. Tell the build system that, yes, you really would like to be
   compiling envoy, as you'll be modifying Envoy:

   ```shell
   export YES_I_AM_OK_WITH_COMPILING_ENVOY=true
   export YES_I_AM_UPDATING_THE_BASE_IMAGES=true
   export ENVOY_COMMIT='-'
   ```

   Building Envoy is slow, and most Ambassador contributors do not
   want to rebuild Envoy, so we require the first two environment
   variables as a safety.

   Setting `ENVOY_COMMIT=-` does 3 things:
    1. Tell it to use whatever is currently checked out in
       `./cxx/envoy/` (instead of checking out a specific commit), so
       that you are free to modify those sources.
    2. Don't try to download a cached build of Envoy from a Docker
       cache (since it wouldn't know which `ENVOY_COMMIT` do download
       the cached build for).
    3. Don't push the build of Envoy to a Docker cache (since you're
       still actively working on it).

### 3. Hacking on Envoy

Modify the sources in `./cxx/envoy/`.

### 4. Building and testing your hacked-up Envoy

- **Build Envoy** with `make update-base`.  Again, this is _not_ a
   quick process.  The build happens in a Docker container; you can
   set `DOCKER_HOST` to point to a powerful machine if you like.

- **Test Envoy** and run with Envoy's test suite (which we don't run
  during normal Ambassador development) by running `make check-envoy`.
  Be warned that Envoy's full **test suite** requires several hundred
  gigabytes of disk space to run.

  Inner dev-loop steps:

   * To run just specific tests, instead of the whole test suite, set
     the `ENVOY_TEST_LABEL` enviornment variable.  For example, to run
     just the unit tests in
     `test/common/network/listener_impl_test.cc`, you should run

     ```shell
     ENVOY_TEST_LABEL='//test/common/network:listener_impl_test' make check-envoy
     ```

   * You can run `make envoy-shell` to get a Bash shell in the Docker
     container that does the Envoy builds.

  Interpreting the test results:

   * Unfortunately, at this time, there are a few expected failures in
     the Envoy test suite; we expect some of the Rate Limit Service
     tests to fail because we patch Envoy to use the old Lyft ("v1")
     gRPC name, but don't patch the tests, which expect the new Envoy
     ("v2") gRPC name.  The failures should all look like:

     ```text
       Expected equality of these values:
       "/envoy.service.ratelimit.v2.RateLimitService/ShouldRateLimit"
         Which is: 0x408155
       ratelimit_request_->headers().Path()->value().getStringView()
         Which is: "/pb.lyft.ratelimit.RateLimitService/ShouldRateLimit"
     ```

   * If you see the following message, don't worry, it's harmless; the
     tests still ran:

     ```text
     There were tests whose specified size is too big. Use the --test_verbose_timeout_warnings command line option to see which ones these are.
     ```

     The message means that the test passed, but it passed too
     quickly, and Bazel is suggesting that you declare it as smaller.
     Something along the lines of "This test only took 2s, but you
     declared it as being in the 60s-300s ('moderate') bucket,
     consider declaring it as being in the 0s-60s ('short')
     bucket".

     Don't be confused (as I was) in to thinking that it was saying
     that the test was too big and was skipped and that you need to
     throw more hardware at it.

- **Build or test Ambassador** with the usual `make` commands, with
  the exception that you MUST run `make update-base` first whenever
  Envoy needs to be recompiled; it won't happen automatically.  So
  `make test` to build-and-test Ambassador would become `make
  update-base && make test`, and `make images` to just build
  Ambassador would become `make update-base && make images`.  By
  default (to keep the tests fast), the tests avoid running much
  traffic through Envoy, and instead just check that the Envoy
  configuration that Ambassador generates hasn't changed since the
  previous version (since we generally trust that Enovy works, and
  doesn't change as aoften).  Since you _are_ changing Envoy, you'll
  need to run the tests with `KAT_RUN_MODE=envoy` set in the
  environment in order to actually test against Envoy.

### 5. Finalizing your changes

Once you're happy with your changes to Envoy:

1. Ensure they're committed to `cxx/envoy/` and push/PR them in to
   https://github.com/datawire/envoy branch `rebase/master`.

   If you're outside of Datawire, you'll need to
    a. Create a fork of https://github.com/datawire/envoy on the
       GitHub web interface
    b. Add it as a remote to your `./cxx/envoy/`:
       `git remote add my-fork git@github.com:YOUR_USERNAME/envoy.git`
    c. Push the branch to that fork:
       `git push my-fork YOUR_BRANCHNAME`

2. Update `ENVOY_COMMIT` in `cxx/envoy.mk`

3. Unset `ENVOY_COMMIT=-` and run a final `make update-base` to
   push a cached build:

   ```shell
   export YES_I_AM_OK_WITH_COMPILING_ENVOY=true
   export YES_I_AM_UPDATING_THE_BASE_IMAGES=true
   unset ENVOY_COMMIT
   make update-base
   ```

   The image will be pushed to `$ENVOY_DOCKER_REPO`, by default
   `ENVOY_DOCKER_REPO=docker.io/datawire/ambassador-base`; if you're
   outside of Datawire, you can skip this step if you don't want to
   share your Envoy binary anywhere.  If you don't skip this step,
   you'll need to `export
   ENVOY_DOCKER_REPO=${your-envoy-docker-registry}` to tell it to push
   somewhere other than Datawire's registry.

   If you're at Datawire, you'll then want to make sure that the image
   is also pushed to the backup container registries:

   ```shell
   TAG=GET_THIS_FROM_THE_make_update-base_OUTPUT

   source_registry=docker.io/datawire
   docker pull "$source_registry/ambassador-base:$TAG
   for target_registry in quay.io/datawire grc.io/datawire; do
     docker tag "$source_registry/ambassador-base:$TAG" "$target_registry/ambassador-base:$TAG"
     docker push "$target_registry/ambassador-base:$TAG"
   done
   ```

   If you're outside of Datawire, you can skip this step if you
   don't want to share your Envoy binary anywhere.  If you don't
   skip this step, you'll need to `export
   ENVOY_DOCKER_REPO=${your-envoy-docker-registry}` to tell it to
   push somewhere other than Datawire's registry.

4. Push/PR the `envoy.mk` `ENVOY_COMMIT` change to
   https://github.com/datawire/ambassador (or
   https://github.com/datawire/apro if you're inside Datawire).

### 6. Checklist for landing the changes

I'd put this in in the pull request template, but so few PRs change Envoy...

 - [ ] The image has been pushed to...
   * [ ] `docker.io/datawire/ambassador-base`
   * [ ] `quay.io/datawire/ambassador-base`
   * [ ] `gcr.io/datawire/ambassador-base`
 - [ ] The envoy.git commit has been tagged as `datawire-$(git
   describe --tags --match='v*')` (the `--match` is to prevent
   `datawire-*` tags from stacking on eachother).
 - [ ] It's been tested with...
   * [ ] `make check-envoy`
   * [ ] `make pytest KAT_RUN_MODE=envoy`

Developing Ambassador (Datawire-only advice)
============================================

At the moment, these techniques will only work internally to Datawire. Mostly
this is because they require credentials to access internal resources at the
moment, though in several cases we're working to fix that.

How do I test my documentation work?
------------------------------------

*This will currently only work within Datawire.*

After you've made some documentation changes, run

```
bash ambassador/scripts/doc-setup
```

to do all the Javascript work needed to get a local documentation server
running. You can point a web browser to `http://localhost:8000` to view docs
with your changes.

After running `ambassador/scripts/doc-setup`, if you need to make further changes, run

```
bash ambassador/scripts/doc-sync
```

to push your changes to the local webserver. They should appear immediately
(you may have to reload the page in your browser).

How do I share a preview of my documentation work with others?
--------------------------------------------------------------

*This will currently only work within Datawire.*

After running `ambassador/scripts/doc-setup`, run

```
bash ambassador/scripts/doc-publish
```

to push a preview to the `getambassador-preview` Netlify site.
Find the Netlify preview URL in the output and hand it off to others.

How do I update the getambassador-preview site with my documentation for others to use?
---------------------------------------------------------------------------------------

*This will currently only work within Datawire.*

After running `ambassador/scripts/doc-setup`, run

```
bash ambassador/scripts/doc-publish --prod
```

to update the live `getambassador-preview` Netlify site.
