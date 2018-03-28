Releasing Ambassador
====================

If you just want to **use** Ambassador, check out https://www.getambassador.io/! You don't need to build - much less release! - anything, and in fact you shouldn't.

If you don't work at Datawire, this document is probably not going to help you. Maybe check out [the developer guide](BUILDING.md) instead.

----

If you're still reading, you must be at Datawire. Congrats, you picked a fine place to work! To release Ambassador, you'll need credentials for our Github repos.

1. Build and test on defect or feature branches. 
2. Merge work into `develop` as you go. Use PRs.
3. Once `develop` has all the release drivers, passes CI, and passes the `end-to-end` tests, open a PR against `master`.
4. The PR had better pass CI. If not, figure out why.
5. Merge the PR to `master`. This kicks off the primary release.
6. **READ THIS WHOLE LINE FOR THE VERY IMPORTANT NOTE.** Once CI runs on the merge commit and the new image has been published, you'll be certain of the new version number and can update the `CHANGELOG`. **VERY IMPORTANT NOTE: commit only `CHANGELOG.md`, and include `[ci skip]` in the commit message** so that you don't bump the version again. Commit and push directly to `master`.
7. Run `make helm` to update helm charts. This will update stuff in `docs`.
8. Edit `docs/index.html` with the new version number.
9. Commit and push directly to `master`.
10. Merge `master` into `develop`.

