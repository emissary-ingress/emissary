# Ambassador Pro [![CircleCI](https://circleci.com/gh/datawire/apro.svg?style=svg&circle-token=81544a8dc30c28df7705975ad2dd4bfee63b653b)](https://circleci.com/gh/datawire/apro)

This is the proprietary Ambassador Pro source-code.  The public
user-facing documentation and issue-tracking lives at
<https://github.com/datawire/ambassador-pro>.

## CI/CD and manual release

### Continuous Integration

On every Git branch, tag, and pull request, CircleCI runs:
 - the `go test` unit tests on Ubuntu and macOS
 - the e2e tests on Ubuntu+Kubernaut+`ambassador-oauth-e2e.auth0.com`

It does NOT push Docker images to any persistent registry for normal
CI runs (it pushes the test images to an ephemeral registry inside of
the Kubernaut cluster).

### Continuous Deployment

On Git tags matching `vX.Y.Z[-PRE]` (for integers `X`, `Y`, `Z`, and
arbitrary string `PRE`), CircleCI does the
[above](#continuous-integration), and (assuming the tests pass),
proceeds to:
 - push `apictl` and `apictl-key` for Linux and Darwin to AWS S3
 - push all 4 Docker images to
   [`quay.io/datawire/ambassador_pro`](https://quay.io/repository/datawire/ambassador_pro?tab=tags)

### Manual release

If you would like to push a development version without tagging a
release or pre-release, you may run

    $ make release

which will build and push a release with the pseudo-version generated
by `git describe --tags`.  You will need the appropriate Quay and AWS
credentials.

## Local development

### Building

Because the `go.mod` specifies dependencies on a private repo
(https://github.com/datawire/liboauth2.git), it is necessary to
configure `git` such that `go get` will be able to fetch it.  That
means sticking this in your `~/.config/git/config`:

    [url "git@github.com:datawire/liboauth2"]
            insteadOf = https://github.com/datawire/liboauth2

Or, I've been happy making that substitution for all GitHub repos:

    [url "git@github.com:"]
            pushInsteadOf = https://github.com/
            insteadOf = https://github.com/

Once you've configured Git, you should be able to build everything by
simply running:

    $ make build

This will build
 - all executable programs, which it will put in
   `./bin_$(go env GOOS)_$(go env GOARCH)/`
 - all Docker images, which it will tag as
   `localhost:31000/$(IMAGE_NAME):$(VERSION)` (or
   `host.docker.internal:31000` on macOS).

### Testing

    $ # if on macOS, first you must configure dockerd, see below
    $ export KUBERNAUT_TOKEN=...
    $ make check

This will run both unit tests and e2e tests.

 > *NOTE:* This will talk to the Auth0 account configured in
 > `./k8s-env.sh`.  The login credentials for that Auth0 account can
 > be found in Keybase under
 > `/datawireio/global/ambassador-oauth-ci.txt`.

On macOS, you will first need to add `host.docker.internal:31000` to
Docker's list of "Insecure registries":

<p align="center">
  <img src="./build-aux/docs/docker-cluster-macos-insecure-registries.png" alt="Docker for Mac &quot;Preferences…&quot; dialog to set the list of &quot;Insecure registries&quot;"/>
</p>

#### Troubleshooting the development environment

    $ make help

Lists all the available targets. See also the [`build-aux` docs](./build-aux/docs/intro.md)

    $ make status

Shows the status of kubernaut cluster claim and status of teleproxy

    $ make claim
    $ make unclaim
    
Manage the claim of the kubernaut cluster. Claims usually expire within a day.

    $ make proxy
    $ make unproxy
    
Start/stop the teleproxy. If proxy does not want to start have a look at `build-aux/teleproxy.log`

## Documentation

The documentation lives in
<https://github.com/datawire/ambassador-docs>, which is included in
this repository at `./docs/` as a `git subtree`.  Functionality
changes that require changes or updates to documentation should have
those documentation changes to `./docs/` included in the PR.

Any new documentation pages that are Pro-only should be mentioned in
`./docs/pro-pages.yml`.

## Cutting an RC or non-publicized release

Simply create a Git tag, and push that Git tag.  e.g.:

    $ git tag v0.1.2-rc3
    $ git push origin v0.1.2-rc3

See [Continuous Deployment](#continuous-deployment) above for
information on what this does, and on the format of the tag names.

## Cutting a GA release

1. ~~Ensure that any Ambassador documentation changes have been merged:~~

        $ make pull-docs           # DON'T DO THIS

   `git subtree` is broken, don't do this for now.

2. Determine the version number. We currently use the `0.x.y` versioning
   scheme. If there is new functionality, we should increment `x`. If
   there are only bug fixes, we should increment `y`.

3. Update the CHANGELOG.md.

4. Tag and push that commit:

        $ git tag v0.1.2
        $ git push origin v0.1.2 master

   See [Continuous Deployment](#continuous-deployment) above for
   information on what this does, and on the format of the tag names.

   This will publish Docker images, `apictl`, and associated
   artifacts, but won't yet publicize it on the website.

5. (this step may be performed before CI for step 3 had finished) Create
   a PR against <https://github.com/datawire/ambassador-docs> that
   updates the `version.yml` with the new version number, and makes
   any other nescessary doc changes.

6. (this step may be performed before CI for step 3 had finished)
   Create a PR against <https://github.com/datawire/pro-ref-arch> that
   updates it for the new version.  This may be as simple as updating
   the version numbers in the several YAML files that mention it.

7. (this step may be performed before CI for step 3 had finished) Create
   a PR against <https://github.com/helm/charts/tree/master/stable/ambassador>
   that updates Pro for the new version.

8. (CI for step 3 must finish before performing this step) Create a PR
   against <https://github.com/datawire/apro-example-plugin> that
   bumps `Makefile:APRO_VERSION` to the new version.  Run `make` to
   verify whether any `go.mod` changes are necessary when updating a
   plugin to the new version.  If `go.mod` changes are necessary, make
   them and include them in the PR.

9. Put the release through manual acceptance testing. We'll do this by
   upgrading `tour.k736.net` to the latest version of Pro, and making
   sure that `tour.k736.net` still functions.

10. ~~From apro.git, with the tag version tag checked out, run `make
    push-docs`:~~

        $ git checkout v0.1.2
        $ make push-docs           # DON'T DO THIS

    `git subtree` is broken, don't do this for now.

11. Ping Noah on Slack to ask him to upgrade our demo cluster to the
    latest version.

12. Merge the `ambassador-docs` PR created in step 4.

13. Merge the `pro-ref-arch` PR created in step 5.

14. Merge the `apro-example-plugin` PR created in step 6.

15. Merge the `helm` PR created in step 7.
