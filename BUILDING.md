Building Ambassador
===================

If you just want to **use** Ambassador, check out the [README.md](README.md)! You don't need to build anything.

If you really want to customize Ambassador, though, read on -- but **NOTE WELL**! This process will change soon.

You'll need the following:

- bash
- make
- docker
- Python 3
- [bump2version](https://pypi.python.org/pypi/bump2version)

(Honestly, you only need Python for `bump2version`.)

Normal Workflow
===============

0. `export DOCKER_REGISTRY=$registry`

   This sets the registry to which to push Docker images and is **mandatory** if you're not using Minikube. The `$registry` info should be the prefix for `docker push`:

   "dwflynn" will push to Dockerhub with user `dwflynn`
   "gcr.io/flynn" will push to GCR with user `flynn`

   You can separately tweak the registry from which images will be _pulled_ using `AMBASSADOR_REGISTRY` and `STATSD_REGISTRY`. See the files in `templates` for more here.

1. Make changes. Commit.

   Hopefully this step is clear.

2. `make new-$level`

   This will correctly set the version number everywhere, then build (and probably push) Docker images, then build YAML files for you. IT WILL NOT COMMIT OR TAG.

   `$level` must be one of "major", "minor", or "patch", using [semantic versioning](http://semver.org/):

   "major" is for major breaking changes.
   "minor" is for new functionality that's still backward compatible.
   "patch" is for bug fixes.

   (You can do "make artifacts" if you want to rebuild artifacts but not change the version, even though that's likely to not be a great idea.)

3. `make tag`

   Do this once you're happy with everything. It will commit (if need be) and then create a Git tag for your version.

   **IT WILL NOT PUSH YOUR COMMIT OR YOUR TAG.** Do that on your own.

What if I Don't Want to Push My Images?
---------------------------------------

**NOTE WELL**: if you're not using Minikube, this is almost certainly a mistake.

But suppose you are using Minikube. The Makefile (deliberately) requires you to set DOCKER_REGISTRY, so you can't just unset it -- instead, set it to "-" to prevent any pushes.
