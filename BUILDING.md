Building Ambassador
===================

If you just want to **use** Ambassador, check out https://www.getambassador.io/! You don't need to build anything, and in fact you shouldn't.

TL;DR
-----

```
git clone https://github.com/datawire/ambassador
cd ambassador
# Development should be on branches based on the `master` branch
git checkout master
git checkout -b username/feature/my-branch-here
make DOCKER_REGISTRY=-
```

That will build an Ambassador Docker image for you but not push it anywhere. To actually push the image into a registry so that you can use it for Kubernetes, set `DOCKER_REGISTRY` to a registry that you have permissions to push to.

**It is important to use `make` rather than trying to just do a `docker build`.** Actually assembling a Docker image for Ambassador involves quite a few steps before the image can be built.

Branching
---------

1. `master` is the branch from which Ambassador releases are cut. There should be no other activity on `master`.

2. **All development must be on branches cut from `master`.** When you have things working and tested, submit a pull request back to `master`.
   - There is a special type of branch for e.g. minor doc changes called `nobuild`. Any branch that matches `^nobuild.*` skips all CI activities for that branch.
   
3. Once your branch is at a point where it's ready for review and testing, open a GitHub PR against `master`.
   - The PR will trigger CI to perform a build and run tests.
   - Tests **must** be passing for the PR to be merged.

4. When all is well, maintainers will merge the PR into `master`.

5. Releases are driven by the maintainers applying tags to `master`.

Documentation Changes
---------------------

Note that documentation changes _still follow the PR process_. If you're making minor changes (fixing typos, for example) it's OK to use a `nobuild` branch as above. If you're doing significant changes, you might want to allow CI to run by using a branch name `doc/major-doc-changes` or the like.

Unit Tests
----------

Unit tests for Ambassador are run on every build. **You are strongly encouraged to add tests when you add features, and you should never ever commit code with failing unit tests.** 

For more information on the unit tests, see [their README](ambassador/tests/README.md).

End-to-End Tests
----------------

Ambassador's end-to-end tests are run by CI for pull requests, release candidates, and releases: we will not release an Ambassador for which the end-to-end tests are failing. **Again, you are strongly encouraged to add end-to-end test coverage for features you add.** 

For more information on the end-to-end tests, see [their README](end-to-end/README.md).

Version Numbering
-----------------

Version numbers will be determined by Git tags for actual releases. You are free to pick whatever version numbers you like when doing test builds.

Normal Workflow
---------------

0. `export DOCKER_REGISTRY=$registry`

   This sets the registry to which to push Docker images and is **mandatory**.

   "dwflynn" will push to Dockerhub with user `dwflynn`
   "gcr.io/flynn" will push to GCR with user `flynn`

   If you're using Minikube and don't want to push at all, set `DOCKER_REGISTRY` to "-".

   You can separately tweak the registry from which images will be _pulled_ using `AMBASSADOR_REGISTRY` and `STATSD_REGISTRY`. See the files in `templates` for more here.

1. Use a private branch cut from `master` for your work.

2. Hack away, then `make`. This will:

   a. Run tests and bail if something doesn't pass.
   b. Build Docker images, and push them if DOCKER_REGISTRY says to.
   c. Build YAML files for you in `doc/yaml`.

   You can easily `kubectl apply` the updated YAML files and see your changes in your Kubernetes cluster.

3. Commit to your feature branch.

   Remember: _not to `master`_.

4. Open a pull request against `master` when you're ready to ship.

What if I Don't Want to Push My Images?
---------------------------------------

**NOTE WELL**: if you're not using Minikube, this is almost certainly a mistake.

But if you are using Minikube, you can set `DOCKER_REGISTRY` to "-" to prevent pushing the images. The Makefile (deliberately) requires you to set DOCKER_REGISTRY, so you can't just unset it.

Building the Documentation and Website
--------------------------------------

Use `make website` to build the docs and website. See [the README](docs/README.md) for docs-specific steps. The `docs/build-website.sh` script (used by `make`) follows those steps and then performs some additional hacks for website use.
