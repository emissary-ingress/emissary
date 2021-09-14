help:
	@printf '%s\n' $(call quote.shell,$(_help.intro))
.PHONY: help

targets:
	@printf '%s\n' $(call quote.shell,$(HELP_TARGETS))
.PHONY: help

define HELP_TARGETS
$(BLD)Targets:$(END)

$(_help.targets)

$(BLD)Codebases:$(END)
  $(foreach MODULE,$(MODULES),$(NL)  $(BLD)$(SOURCE_$(MODULE)) ==> $(BLU)$(MODULE)$(END))

endef

# Style note: _help.intro
# - is wrapped to 72 columns (after stripping the ANSI color codes)
# - has sentences separated with 2 spaces
# - uses bold blue ("$(BLU)") when introducing a new variable
# - uses bold ("$(BLD)") for variables that have already been introduced
# - uses bold ("$(BLD)") when you would use `backticks` in markdown
define _help.intro
This Makefile builds Ambassador using a standard build environment
inside a Docker container.  The $(BLD)$(REPO)$(END), $(BLD)kat-server$(END), and $(BLD)kat-client$(END)
images are created from this container after the build stage is
finished.

The build works by maintaining a running build container in the
background.  It gets source code into that container via $(BLD)rsync$(END).  The
$(BLD)/home/dw$(END) directory in this container is a Docker volume, which allows
files (e.g. the Go build cache and $(BLD)pip$(END) downloads) to be cached across
builds.

This arrangement also permits building multiple codebases.  This is
useful for producing builds with extended functionality.  Each external
codebase is synced into the container at the $(BLD)/buildroot/<name>$(END) path.

You can control the name of the container and the images it builds by
setting $(BLU)$$BUILDER_NAME$(END), which defaults to $(BLD)$(LCNAME)$(END).  Note well that if
you want to make multiple clones of this repo and build in more than one
of them at the same time, you $(BLD)must$(END) set $(BLD)$$BUILDER_NAME$(END) so that each clone
has its own builder!  If you do not do this, your builds will collide
with confusing results.

The build system doesn't try to magically handle all dependencies.  In
general, if you change something that is not pure source code, you will
likely need to do a $(BLD)$(MAKE) clean$(END) in order to see the effect.  For example,
Python code only gets set up once, so if you change $(BLD)setup.py$(END), then you
will need to do a clean build to see the effects.  Assuming you didn't
$(BLD)$(MAKE) clobber$(END), this shouldn't take long due to the cache in the Docker
volume.

All targets that deploy to a cluster by way of $(BLU)$$DEV_REGISTRY$(END) can be made
to have the cluster use an imagePullSecret to pull from $(BLD)$$DEV_REGISTRY$(END),
by setting $(BLU)$$DEV_USE_IMAGEPULLSECRET$(END) to a non-empty value.  The
imagePullSecret will be constructed from $(BLD)$$DEV_REGISTRY$(END),
$(BLU)$$DOCKER_BUILD_USERNAME$(END), and $(BLU)$$DOCKER_BUILD_PASSWORD$(END).

By default, the base builder image is (as an optimization) pulled from
$(BLU)$$BASE_REGISTRY$(END) instead of being built locally; where $(BLD)$$BASE_REGISTRY$(END)
defaults to $(BLD)$$DEV_REGISTRY$(END) or else $(BLD)$${BUILDER_NAME}.local$(END).  If that pull
fails (as it will if trying to pull from a $(BLD).local$(END) registry, or if the
image does not yet exist), then it falls back to building the base image
locally.  If $(BLD)$$BASE_REGISTRY$(END) is equal to $(BLD)$$DEV_REGISTRY$(END), then it will
proceed to push the built image back to the $(BLD)$$BASE_REGISTRY$(END).

Use $(BLD)$(MAKE) $(BLU)targets$(END) for help about available $(BLD)make$(END) targets.
endef

define _help.targets
  $(BLD)$(MAKE) $(BLU)help$(END)         -- displays the main help message.

  $(BLD)$(MAKE) $(BLU)targets$(END)      -- displays this message.

  $(BLD)$(MAKE) $(BLU)env$(END)          -- display the value of important env vars.

  $(BLD)$(MAKE) $(BLU)export$(END)       -- display important env vars in shell syntax, for use with $(BLD)eval$(END).

  $(BLD)$(MAKE) $(BLU)preflight$(END)    -- checks dependencies of this makefile.

  $(BLD)$(MAKE) $(BLU)sync$(END)         -- syncs source code into the build container.

  $(BLD)$(MAKE) $(BLU)version$(END)      -- display source code version.

  $(BLD)$(MAKE) $(BLU)compile$(END)      -- syncs and compiles the source code in the build container.

  $(BLD)$(MAKE) $(BLU)images$(END)       -- creates images from the build container.

  $(BLD)$(MAKE) $(BLU)push$(END)         -- pushes images to $(BLD)$$DEV_REGISTRY$(END). ($(DEV_REGISTRY))

  $(BLD)$(MAKE) $(BLU)test$(END)         -- runs Go and Python tests inside the build container.

    The tests require a Kubernetes cluster and a Docker registry in order to
    function. These must be supplied via the $(BLD)$(MAKE)$(END)/$(BLD)env$(END) variables $(BLD)$$DEV_KUBECONFIG$(END)
    and $(BLD)$$DEV_REGISTRY$(END).

  $(BLD)$(MAKE) $(BLU)gotest$(END)       -- runs just the Go tests inside the build container.

    Use $(BLD)$$GOTEST_PKGS$(END) to control which packages are passed to $(BLD)gotest$(END). ($(GOTEST_PKGS))
    Use $(BLD)$$GOTEST_ARGS$(END) to supply additional non-package arguments. ($(GOTEST_ARGS))
    Example: $(BLD)$(MAKE) gotest GOTEST_PKGS=./cmd/entrypoint GOTEST_ARGS=-v$(END)  # run entrypoint tests verbosely

  $(BLD)$(MAKE) $(BLU)pytest$(END)       -- runs just the Python tests inside the build container.

    Use $(BLD)$$KAT_RUN_MODE=envoy$(END) to force the Python tests to ignore local caches, and run everything
    in the cluster.

    Use $(BLD)$$KAT_RUN_MODE=local$(END) to force the Python tests to ignore the cluster, and only run tests
    with a local cache.

    Use $(BLD)$$PYTEST_ARGS$(END) to pass args to $(BLD)pytest$(END). ($(PYTEST_ARGS))

    Example: $(BLD)$(MAKE) pytest KAT_RUN_MODE=envoy PYTEST_ARGS="-k Lua"$(END)  # run only the Lua test, with a real Envoy

  $(BLD)$(MAKE) $(BLU)pytest-gold$(END)  -- update the gold files for the pytest cache

    $(BLD)$(MAKE) $(BLU)pytest$(END) uses a local cache to speed up tests. $(BLD)ONCE YOU HAVE SUCCESSFULLY
    RUN TESTS WITH $(BLU)KAT_RUN_MODE=envoy$(END), you can use $(BLD)$(MAKE) $(BLU)pytest-gold$(END) to update the
    caches for the passing tests.

    $(BLD)DO NOT$(END) run $(BLD)$(MAKE) $(BLU)pytest-gold$(END) if you have failing tests.

  $(BLD)$(MAKE) $(BLU)shell$(END)        -- starts a shell in the build container

    The current commit must be tagged for this to work, and your tree must be clean.
    Additionally, the tag must be of the form 'vX.Y.Z-rc.N'. You must also have previously
    built an RC for the same tag using $(BLD)release/bits$(END).

  $(BLD)$(MAKE) $(BLU)release/promote-oss/to-ga$(END) -- promote a release candidate to general availability

    The current commit must be tagged for this to work, and your tree must be clean.
    Additionally, the tag must be of the form 'vX.Y.Z'. You must also have previously
    built and promoted the RC that will become GA, using $(BLD)release/bits$(END).

  $(BLD)$(MAKE) $(BLU)clean$(END)     -- kills the build container.

  $(BLD)$(MAKE) $(BLU)clobber$(END)   -- kills the build container and the cache volume.

  $(BLD)$(MAKE) $(BLU)generate$(END)  -- update generated files that get checked in to Git.

    1. Use $(BLD)$$ENVOY_COMMIT$(END) to update the vendored gRPC protobuf files ('api/envoy').
    2. Run 'protoc' to generate things from the protobuf files (both those from
       Envoy, and those from 'api/kat').
    3. Use $(BLD)$$ENVOY_GO_CONTROL_PLANE_COMMIT$(END) to update the vendored+patched copy of
       envoyproxy/go-control-plane ('pkg/envoy-control-plane/').
    4. Use the Go CRD definitions in 'pkg/api/getambassador.io/' to generate YAML
       (and a few 'zz_generated.*.go' files).

  $(BLD)$(MAKE) $(BLU)update-yaml$(END) -- like $(BLD)make generate$(END), but skips the slow Envoy stuff.

  $(BLD)$(MAKE) $(BLU)go-mod-tidy$(END) -- 'go mod tidy', but plays nice with 'make generate'

  $(BLD)$(MAKE) $(BLU)guess-envoy-go-control-plane-commit$(END) -- Make a suggestion for setting ENVOY_GO_CONTROL_PLANE_COMMIT= in generate.mk

  $(BLD)$(MAKE) $(BLU)lint$(END)        -- runs golangci-lint.

  $(BLD)$(MAKE) $(BLU)format$(END)      -- runs golangci-lint with --fix.

endef
