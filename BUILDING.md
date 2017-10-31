Building Ambassador
===================

If you just want to **use** Ambassador, check out https://www.getambassador.io/! You don't need to build anything, and in fact you shouldn't.

If you really want to customize Ambassador, though, read on.

- bash
- make
- docker
- Python 3

Version Numbering
=================

**Version numbers are determined by tags in Git, and will be computed by the build process.**

This means that if you build repeatedly without committing, you'll get the same version number. This isn't a problem as you debug and hack, although you may need to set `imagePullPolicy: Always` to have things work smoothly.

It also means that we humans don't say things like "I'm going to make this version 1.23.5" -- Ambassador uses [Semantic Versioning](http://www.semver.org/), and the build process computes the next version by looking at changes since the last tagged version. Start a Git commit comment with `[MINOR]` to tell the build that the change is worthy of a minor-version bump; start it with `[MAJOR]` for a major-version bump. If no marker is present, the patch version will be incremented.

Normal Workflow
===============

0. `export DOCKER_REGISTRY=$registry`

   This sets the registry to which to push Docker images and is **mandatory**.

   "dwflynn" will push to Dockerhub with user `dwflynn`
   "gcr.io/flynn" will push to GCR with user `flynn`

   If you're using Minikube and don't want to push at all, set `DOCKER_REGISTRY` to "-".

   You can separately tweak the registry from which images will be _pulled_ using `AMBASSADOR_REGISTRY` and `STATSD_REGISTRY`. See the files in `templates` for more here.

1. Be on a branch that isn't `master`.

   At Datawire, committing to `master` triggers the CI pipeline to do an actual release. Even if you're not at Datawire, it's better to do you Ambassador development on a feature branch.

2. Hack away, then `make`. This will:

   a. Compute a version number based on git tags (`git describe --tags`).
   b. Push that version number everywhere in the code that it needs to be.
   c. Run tests and bail if something doesn't pass.
   d. Build Docker images, and push them if DOCKER_REGISTRY says to.
   e. Build YAML files for you in `doc/yaml`.

   **IT WILL NOT COMMIT OR TAG**. With new version numbers everywhere, you can easily `kubectl apply` the updated YAML files and see your changes in your Kubernetes cluster.

   If you make further changes and `make` again, _the version number will not change_. To get a new version number, you'll need to commit.

3. Commit to your feature branch.

   Remember: _not to `master`_.

4. Open a pull request against `master` when you're ready to ship.

What if I Don't Want to Push My Images?
---------------------------------------

**NOTE WELL**: if you're not using Minikube, this is almost certainly a mistake.

But if you are using Minikube, you can set `DOCKER_REGISTRY` to "-" to prevent pushing the images. The Makefile (deliberately) requires you to set DOCKER_REGISTRY, so you can't just unset it.

Tests
=====

Unit tests for Ambassador live in the `ambassador/tests` directory. Within that directory, you'll find directories with names that start with three digits (`000-default`, `001-broader-v0`, etc); each of these directories describes a test environment.

Each test directory MUST contain either a `config` directory or the magic `TEST_DEFAULT_CONFIG` marker, which says to use `ambassador/default-config` for configuration. Test directories MAY also contain a `gold.json`, and other files that need to be referenced by the Ambassador configuration (for example, TLS certificates -- see `001-broader-v0` for an example).

For each test directory:

1. The test starts by using the Ambassador configuration for the test to generate an `envoy.json`.
2. That `envoy.json` is then handed to Envoy for validation -- this step runs inside Docker, and the test directory is mounted inside the Docker container as `/etc/ambassador-config`.
3. If a `gold.json` is present, the `envoy.json` is compared to the `gold.json` and must match.
4. Finally, Ambassador's diagnostic service is invoked on the Ambassador configuration, and the results are checked for internal consistency.

You can add new tests simply by adding more test directories under `ambassador/tests`.

Building the documentation and website
======================================

Use `make website` to build the docs and website. See [the README](docs/README.md) for docs-specific steps. The `docs/build-website.sh` script (used by `make`) follows those steps and then performs some additional hacks for website use.
