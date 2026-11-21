Releasing Ambassador
====================

If you just want to **use** Ambassador, check out https://www.getambassador.io/! You don't need to build - much less release! - anything, and in fact you shouldn't.

If you don't work at Datawire, this document is probably not going to help you. Maybe check out [the developer guide](BUILDING.md) instead.

----

If you're still reading, you must be at Datawire. Congrats, you picked a fine place to work! To release Ambassador, you'll need credentials for our Github repos.

Note. PRs will pile up on `master`. **Don't accept PRs for which CI doesn't show passing tests.**
When we get to the stage of creating a release, all the PRs that we want to merge will have been merged
and the CI will be green.

1. Once `master` has all the release drivers, tag `master` with an RC tag, e.g. `v0.77.0-rc.1`. **This version tag must start with a 'v'.** For example:
    git tag v0.77.0-rc.1 master
    git push --tags origin master

2. The RC tag will trigger CI to run a new build and new tests. It had better pass: if not, figure out why. Monitor https://travis-ci.com/datawire/amabassador/ until the CI for ambassador completes and is green.

3. The RC build will be available as e.g. `docker.io/datawire/ambassador:0.77.0-rc.1`. Any other testing you want to do against this image, rock on.

4. When you're happy that the RC is ready for GA, review the changelog snippets in `docs/changes/` (see `docs/changes/README.md`).
   Every user-visible change should already have a snippet, added by its PR. Preview the release notes with
   `uv run --dev -- towncrier build --draft --version X.Y.Z`, and fix up any snippets that need it:
   - We always call out things contributed by the community, including who committed it
     - you can mention the contributor with a link back to their GitHub page
   - We always call out features and major bugfixes
   - We always include links to GitHub issues where we can (a snippet's file name supplies its issue link)
   - We rarely include fixes to the tests, build infrastructure, etc.
   - Cross-check against `git log v0.77.0..HEAD --no-merges` and the list of closed PRs for anything that is
     missing a snippet.

5. Once the release notes look right, hand them to Marketing so they can either write a blog post or tell you it's not needed.

6. While the blog post is being written, switch to a new branch for the release.
   - `git checkout -b rel/0.84.0` (or whatever version). **Do not name this branch "release/...".**

7. Use `make release/changelog-update CHANGELOG_VERSION=X.Y.Z` (or `X.Y.Z-rc.N` for an RC) to update `CHANGELOG.md`.
   - This runs towncrier, which assembles the snippets in `docs/changes/` into a new section of `CHANGELOG.md`, replacing any earlier RC sections for the same version. It deletes the snippets for a GA, but keeps them for an RC.
   - It does not commit. Make sure the diff looks correct, then `git add -A CHANGELOG.md docs/changes`, commit, and `git push` your branch.

8. Now for the time-critical bit.
   - Tag your branch with a GA tag like `v0.77.0` and let CI do its thing. **This version tag must start with a 'v'.**
   - CI will retag the latest RC image as the GA image.

9. _After the CI run finishes_:
   - Submit, and land, a PR for your branch.
   - Go to https://github.com/datawire/ambassador/releases and create a new release.
      - The new `CHANGELOG.md` section has the text to paste into the release description.

   **Note** that there must be at least one RC build before a GA, since the GA tag **does not** rebuild the docker images -- it retags the ones built by the RC build. This is intentional, to allow for testing to happen on the actual artifacts that will be released.

   **Note also** that even though the version _tag_ starts with a 'v', the version _number_ displayed by the diag UI will _not_ start with a 'v'.**

10. Submit a PR into the `helm/charts` repo to update things in `stable/ambassador`:
   - in `Chart.yaml`, update `appVersion` with the new Ambassador version, and bump `version`.
   - in `README.md`, update the default value of `image.tag`.
   - in `values.yaml`, update `tag`.
   - Helpful stuff for this:
      - git checkout master               # switch to master
      - git fetch --all --prune           # make sure our view of remotes is up to date
      - git pull                          # pull down any changes to master
      - git rebase upstream/master        # move master on top of upstream
      - git push                          # push rebases to our fork
      - git checkout -b update/$VERSION   # switch to a feature branch
      - make your edits
      - git commit -a                     # commit changes -- don't forget DCO in the message!
      - git push origin update/$VERSION   # push to feature branch
      - open a PR
    - Once your PR is merged, _delete the feature branch without merging it back to origin/master_.

11. Update the getambassador.io website by submitting a PR to the `datawire/getambassador.io` repo.
   - `src/releaseInfo.js` is the only file you should need to touch:
      - `ReleaseType` comes from Marketing, usually "Feature Release", "Maintenance Release", or "Security Update"
      - `CurrentVersion` is e.g. "0.78.0" -- no leading 'v' here
      - `BlogLink` is the full URL of the blog post (from Marketing), or "" if there is no blog post
   - Make your edits, submit a PR, get it merged. Done.
      - If you want to test before submitting, use `yarn install && yarn start` and point a web browser to `localhost:8000`

   Submit a PR to the Ambassador website repository to update the version on the homepage. Details can be found there on specifically how to do that.
