Building Ambassador
===================

If you just want to **use** Ambassador, check out http://www.getambassador.io/! You don't need to build anything, and in fact you shouldn't.

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

   This sets the registry to which to push Docker images and is **mandatory** if you're not using Minikube. The `$registry` info should be the prefix for `docker push`, for example:

   "dwflynn" will push to Dockerhub with user `dwflynn`
   "gcr.io/flynn" will push to GCR with user `flynn`

   You can separately tweak the registry from which images will be _pulled_ using `AMBASSADOR_REGISTRY` and `STATSD_REGISTRY`. See the files in `templates` for more here.

1. Hack away, then `make`.

   This will compute a version number based on git tags (`git describe --tags`), then push that version number everywhere it needs to be in the code. Then it will build (and probably push) Docker images, then build YAML files for you. **IT WILL NOT COMMIT OR TAG**. With new version numbers everywhere, you can easily `kubectl apply` the updated YAML files and see your changes in your Kubernetes cluster.

   If you make further changes and `make` again, _the version number will not change_. To get a new version number, you'll need to commit.

2. Commit, then `make tag` to mark a version.

   Generally you would do this when you're ready to ship, but it will also be very important if you're working with multiple developers!

   Note that just typing `make` will build a development version, marked with a build number (e.g. `0.10.4-b3.56d8917`). This is intentional: at Datawire, the CI pipeline is the only thing that builds non-development versions.

What if I Don't Want to Push My Images?
---------------------------------------

**NOTE WELL**: if you're not using Minikube, this is almost certainly a mistake.

But suppose you are using Minikube. The Makefile (deliberately) requires you to set DOCKER_REGISTRY, so you can't just unset it -- instead, set it to "-" to prevent any pushes.

Building the documentation and website
======================================

Use `make website` to build the docs and website. See [the README](docs/README.md) for docs-specific steps. The `docs/build-website.sh` script (used by `make`) follows those steps and then performs some additional hacks for website use.
