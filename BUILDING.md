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

* The current release of Ambassador lives on `stable`.

* Development on the next upcoming release of Ambassador happens on `master` (which is the default branch when you clone Ambassador).

### Making Code Changes

1. **All development must be on branches cut from `master`**. 
   - The name of the branch isn't all that relevant, except:
   - A branch whose name starts with `nobuild.` will skip CI activities. This is intended for e.g. minor doc changes.

2. If your development takes any significant time, **merge master back into your branch regularly**.
   - Think "every morning" and "right before submitting a pull request."

3. When you have things working and tested, **submit a pull request back to `master`**.
   - Make sure you merge `master` _into_ your branch right before submitting the PR!
   - The PR will trigger CI to perform a build and run tests.
   - Tests **must** be passing for the PR to be merged.

4. When all is well, maintainers will merge the PR into `master`.

5. Maintainers will PR from `master` into `stable` for releases.

### Making Documentation Changes

1. Documentation changes happen on branches cut from **`stable`**, not `master`.

2. It's OK for minor doc changes (fixing typos, etc) to use a `nobuild.` branch. If you're doing significant changes, let CI run so you can preview the docs -- use a branch name like `doc/major-doc-changes` or the like.

3. **If you have a doc change open when we do a release, it's your job to merge from `stable` into your doc branch.** You should avoid this.

4. When you have things edited as you like them, **submit a PR back to `stable`**.

5. Maintainers (code or doc) will merge the PR to `stable`, then merge the changes from `stable` back into `master`.

Developer Quickstart/Inner Loop
-------------------------------

### Quickstart:

1. git clone ...
2. From git root type `make shell`
3. Run `py.test -k tests_i_care_about`
4. Edit code.
5. Go back to 3.

### Details:

Note that `make shell` will do a bunch of setup the first time it
runs, but subsequent runs should be instantaneous.

You can create as many dev shells as you want. They will all share the
same kubernaut cluster and teleproxy session behind the scenes.

The first time you run the test_ambassador suite it will apply a bunch
of yaml to kubernaut cluster used by your dev session. Be patient,
this will be much faster the second time.

If you want to release the kubernaut cluster and kill teleproxy, then
run `make clean-test`. This will happen automatically when you run
`make clean`.

If you change networks and your dns configuration changes, you will
need to restart teleproxy. You can do this with the `make
teleproxy-restart` target.

Type Hinting
------------

Ambassador uses Python 3 type hinting to help find bugs before runtime. We will not
accept changes that aren't hinted -- if you haven't worked with hinting before, a good
place to start is [the `mypy` cheat sheet](https://mypy.readthedocs.io/en/latest/cheat_sheet_py3.html).

We **strongly** recommend that you use an editor that supports realtime type checking:
we at Datawire tend to use PyCharm and VSCode a lot, but many many editors can do this 
now. We also **strongly** recommend that you run `mypy` itself over your code before 
opening a PR. The easy way to do that is simply

```make mypy```

after you've done `make shell`. This will start the [mypy daemon](https://mypy.readthedocs.io/en/latest/mypy_daemon.html)
and then do a check of all the Ambassador code. There _should_ be no errors and no warnings
reported: that will probably be a requirement for all GA releases.
 
**Note well** that at present, `make mypy` will ignore missing imports. We're still sorting
out how to best wrangle the various third-party libraries we use, so this seems to make sense
for right now -- suggestions welcome on this front!   

Unit Tests
----------

Unit tests for Ambassador are run on every build. **You will be asked to add tests when you add features, and you should never ever commit code with failing unit tests.** 

For more information on the unit tests, see [their README](ambassador/tests/README.md).

End-to-End Tests
----------------

Ambassador's end-to-end tests are run by CI for pull requests, release candidates, and releases: we will not release an Ambassador for which the end-to-end tests are failing. **Again, you will be asked to add end-to-end test coverage for features you add.** 

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

   You can separately tweak the registry from which images will be _pulled_ using `AMBASSADOR_REGISTRY`. See the files in `templates` for more here.

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
