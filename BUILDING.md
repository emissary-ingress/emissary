Building Ambassador
===================

If you just want to **use** Ambassador, check out https://www.getambassador.io/! You don't need to build anything, and in fact you shouldn't.

TL;DR for Code Changes
----------------------

If you're making a code change:

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

If you want to make a doc change, see the `Making Documentation-Only Changes` section below.

### Structure and Branches

The current shipping release of Ambassador lives here on the `master` branch. It is tagged with its version (e.g. `0.52.0`). 

Changes on `master` after the last tag have not been released yet, but will be included in the next release of Ambassador.

The documentation in the `docs` directory is actually a Git subtree from the `ambassador-docs` repo. See the `Making Documentation-Only Changes` section below if you just want to change docs.

### Making Code Changes

1. **All development must be on branches cut from `master`**. 
   - The name of the branch isn't all that relevant, except:
   - A branch whose name starts with `nobuild.` will skip CI activities. This is intended for e.g. minor doc changes.

2. If your development takes any significant time, **merge master back into your branch regularly**.
   - Think "every morning" and "right before submitting a pull request."

3. **Code changes must include relevant documentation updates.** Make changes in
   the `docs` directory as necessary, and commit them to your branch so that they
   can be incorporated when the feature is merged into `master`.

4. **Code changes must include tests.** See `tests/README.md` for more here.
   Your test **must** actually test the change you're making, and it **must**
   pass in order for your change to be accepted.

5. When you have things working and tested, **submit a pull request back to `master`**.
   - Make sure you merge `master` _into_ your branch right before submitting the PR!
   - The PR will trigger CI to perform a build and run tests.
   - CI tests **must** be passing for the PR to be merged.

6. When all is well, maintainers will merge the PR into `master`, accepting your
   change for the next Ambassador release. Thanks!

### Making Documentation-Only Changes

If you want to make a change that **only** affects documentation, and is not 
tied to a future feature, you'll need to make your change directly in the
`datawire/ambassador-docs` repository. Clone that repository and check out
its `README.md`. 

(It is technically possible to make these changes from the `ambassador` repo. Please don't, unless you're fixing docs for an upcoming feature that hasn't yet
shipped.)

Developer Quickstart/Inner Loop
-------------------------------

### Quickstart:

1. `git clone https://github.com/datawire/ambassador`
2. `cd ambassador`
2. `make shell`
3. Run `py.test -k tests_i_care_about`
4. Edit code.
5. Go back to 3.

### Details:

Note that `make shell` will do a bunch of setup the first time it
runs, but subsequent runs should be instantaneous.

You can create as many dev shells as you want. They will all share the
same kubernaut cluster and teleproxy session behind the scenes.

The first time you run the `test_ambassador` suite it will apply a bunch
of YAML to the kubernaut cluster used by your dev session. Be patient,
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
reported: that will probably become a requirement for all GA releases later.
 
**Note well** that at present, `make mypy` will ignore missing imports. We're still sorting
out how to best wrangle the various third-party libraries we use, so this seems to make sense
for right now -- suggestions welcome on this front!   

Tests
-----

CI runs Ambassador's test suite on every build. **You will be asked to add tests when you add features, and you should never ever commit code with failing unit tests.** 

For more information on the test suite, see [its README](ambassador/tests/README.md).

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

2. Hack away, then `make`. This will build Docker images, push if needed, then run the tests.

3. Commit to your feature branch.

   Remember: _not to `master`_.

4. Open a pull request against `master` when you're ready to ship.

What if I Don't Want to Push My Images?
---------------------------------------

**NOTE WELL**: if you're not using Minikube, this is almost certainly a mistake.

But if you are using Minikube, you can set `DOCKER_REGISTRY` to "-" to prevent pushing the images. The Makefile (deliberately) requires you to set DOCKER_REGISTRY, so you can't just unset it.
