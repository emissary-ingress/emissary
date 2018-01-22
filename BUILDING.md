Building Ambassador
===================

If you just want to **use** Ambassador, check out https://www.getambassador.io/! You don't need to build anything, and in fact you shouldn't.

TL;DR
-----

```
git clone https://github.com/datawire/ambassador
# Development should be on branches based on the `develop` branch
git checkout develop
git checkout -b dev/my-branch-here
cd ambassador
# best to activate a Python 3 virtualenv here!
pip install -r dev-requirements.txt
DOCKER_REGISTRY=- make
```

That will build an Ambassador Docker image for you but not push it anywhere. To actually push the image into a registry so that you can use it for Kubernetes, set `DOCKER_REGISTRY` to a registry that you have permissions to push to.

**It is important to use `make` rather than trying to just do a `docker build`.** Actually assembling a Docker image for Ambassador involves quite a few steps before the image can be built.

Branching
---------

1. `master` is the branch from which Ambassador releases are cut. There should be no other activity on `master`.

2. `develop` is the default branch for the repo, and is the base from which all development should happen. 

3. **All development should be on branches cut from `develop`.** When you have things working and tested, submit a pull request back to `develop`.

This implies that `master` must _always_ be shippable and tagged, and `develop` _should_ always be shippable.

Unit Tests
----------

Unit tests for Ambassador live in the `ambassador/tests` directory. Within that directory, you'll find directories with names that start with three digits (`000-default`, `001-broader-v0`, etc); each of these directories describes a test environment.

**You are strongly encouraged to add tests when you add features.** Adding a new test is simply a matter of adding a new test directory under `ambassador/tests`.

Each test directory MUST contain either a `config` directory or the magic `TEST_DEFAULT_CONFIG` marker, which says to use `ambassador/default-config` for configuration. Test directories MAY also contain a `gold.json`, and other files that need to be referenced by the Ambassador configuration (for example, TLS certificates -- see `001-broader-v0` for an example).

For each test directory:

1. The test starts by using the Ambassador configuration for the test to generate an `envoy.json`.
2. That `envoy.json` is then handed to Envoy for validation -- this step runs inside Docker, and the test directory is mounted inside the Docker container as `/etc/ambassador-config`.
3. If a `gold.json` is present, the `envoy.json` is compared to the `gold.json` and must match.
4. Finally, Ambassador's diagnostic service is invoked on the Ambassador configuration, and the results are checked for internal consistency.

Version Numbering
-----------------

**Version numbers are determined by tags in Git, and will be computed by the build process.**

This means that if you build repeatedly without committing, you'll get the same version number. This isn't a problem as you debug and hack, although you may need to set `imagePullPolicy: Always` to have things work smoothly.

It also means that we humans don't say things like "I'm going to make this version 1.23.5" -- Ambassador uses [Semantic Versioning](http://www.semver.org/), and the build process computes the next version by looking at changes since the last tagged version. Start a Git commit comment with `[MINOR]` to tell the build that the change is worthy of a minor-version bump; start it with `[MAJOR]` for a major-version bump. If no marker is present, the patch version will be incremented.

Normal Workflow
---------------

0. `export DOCKER_REGISTRY=$registry`

   This sets the registry to which to push Docker images and is **mandatory**.

   "dwflynn" will push to Dockerhub with user `dwflynn`
   "gcr.io/flynn" will push to GCR with user `flynn`

   If you're using Minikube and don't want to push at all, set `DOCKER_REGISTRY` to "-".

   You can separately tweak the registry from which images will be _pulled_ using `AMBASSADOR_REGISTRY` and `STATSD_REGISTRY`. See the files in `templates` for more here.

1. Use a private branch cut from `develop` for your work.

   Committing to `master` triggers the CI pipeline to do an actual release. The base branch for development work is `develop` -- and you should always create a new branch from `develop` for your changes.

2. Hack away, then `make`. This will:

   a. Compute a version number based on git tags (`git describe --tags`).
   b. Push that version number everywhere in the code that it needs to be.
   c. Run tests and bail if something doesn't pass.
   d. Build Docker images, and push them if DOCKER_REGISTRY says to.
   e. Build YAML files for you in `doc/yaml`.

   **IT WILL NOT COMMIT OR TAG**. With new version numbers everywhere, you can easily `kubectl apply` the updated YAML files and see your changes in your Kubernetes cluster.

   If you make further changes and `make` again, _the version number will not change_. To get a new version number, you'll need to commit.

3. Commit to your feature branch.

   Remember: _not to `develop` or `master`_.

4. Open a pull request against `develop` when you're ready to ship.

What if I Don't Want to Push My Images?
---------------------------------------

**NOTE WELL**: if you're not using Minikube, this is almost certainly a mistake.

But if you are using Minikube, you can set `DOCKER_REGISTRY` to "-" to prevent pushing the images. The Makefile (deliberately) requires you to set DOCKER_REGISTRY, so you can't just unset it.

Building the Documentation and Website
--------------------------------------

Use `make website` to build the docs and website. See [the README](docs/README.md) for docs-specific steps. The `docs/build-website.sh` script (used by `make`) follows those steps and then performs some additional hacks for website use.
