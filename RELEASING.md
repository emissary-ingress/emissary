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

3. The RC build will be available as e.g. `quay.io/datawire/ambassador:0.77.0-rc.1` and also as e.g. `quay.io/datawire/ambassador:0.77.0-rc-latest`. Any other testing you want to do against this image, rock on.

4. When you're happy that the RC is ready for GA, **first** assemble the list of changes that you'll put into CHANGELOG.md: (Note: place this list in a separate file, maybe `~/temp-list.txt`, but definitely not in CHANGELOG.md at this time.
   - We always call out things contributed by the community, including who committed it
     - you can mention the contributor with a link back to their GitHub page
   - We always call out features and major bugfixes
   - We always include links to GitHub issues where we can
   - We rarely include fixes to the tests, build infrastructure, etc.
   - Look at e.g. `git log v0.77.0..HEAD --no-merges --pretty '%h (%ai, %ae, %an): %s' -- ':(exclude)docs'`
     and at the list of closed PRs. This is an awkward area of the release process as there are a log commits
     and PRs but we only want to include a curated subset that makes sense to the users.

5. Once the change list is assembled, hand it to Marketing so they can either write a blog post or tell you it's not needed.

6. While the blog post is being written, switch to a new branch for the release.
   - `git checkout -b rel/0.84.0` (or whatever version). **Do not name this branch "release/...".**

7. Next, sync up docs.
   - `make pull-docs` to pull updates from the docs repo
   - Handle conflicts as needed.
   - Commit any conflict-resolution changes to your release branch.

8. After the docs are synced, use `make release-prep` to update `CHANGELOG.md` and `docs/versions.yml`.
   - This will prompt you for the release notes, so retrieve them from your previous file (maybe `~/temp-list.txt`).
     The release notes are pasted in at a prompt (during the make), not read from a file, so you will need them
     accessible to select-and-copy (suggestion: open your previous file in another window).
   - It will _commit_, but not _push_, the updated files. Make sure the diffs it shows you look correct!
      - It is *critical* to update `docs/versions.yml` so that everyone gets the new version.
   - Do a manual `git push` on your branch.

9. Now for the time-critical bit.
   - Tag your branch with a GA tag like `v0.77.0` and let CI do its thing. **This version tag must start with a 'v'.**
   - CI will retag the latest RC image as the GA image.

10. _After the CI run finishes_:
   - Submit, and land, a PR for your branch.
   - `make push-docs` from `master` _after the retag_ to push new docs out to the website.
   - Go to https://github.com/datawire/ambassador/releases and create a new release.
      - `make release-prep` should've output the text to paste into the release description.

   **Note** that there must be at least one RC build before a GA, since the GA tag **does not** rebuild the docker images -- it retags the ones built by the RC build. This is intentional, to allow for testing to happen on the actual artifacts that will be released.

   **Note also** that even though the version _tag_ starts with a 'v', the version _number_ displayed by the diag UI will _not_ start with a 'v'.**

11. Submit a PR into the `helm/charts` repo to update things in `stable/ambassador`:
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

12. Update the getambassador.io website by submitting a PR to the `datawire/getambassador.io` repo.
   - `src/releaseInfo.js` is the only file you should need to touch:
      - `ReleaseType` comes from Marketing, usually "Feature Release", "Maintenance Release", or "Security Update"
      - `CurrentVersion` is e.g. "0.78.0" -- no leading 'v' here
      - `BlogLink` is the full URL of the blog post (from Marketing), or "" if there is no blog post
   - Make your edits, submit a PR, get it merged. Done.
      - If you want to test before submitting, use `npm install && npm start` and point a web browser to `localhost:8000`

   Submit a PR to the Ambassador website repository to update the version on the homepage.

---

### Host a release branch on getambassador.io

getambassador.io can host multiple versions of ambassador documentation. As a matter of policy, only the documentation for major and minor releases is hosted, documentation changes for patch releases are expected to be folded in the associated minor release.

After a new major/minor release is cut, this is how to host it on the website.
For example, let's suppose version 1.4 of ambassador needs to be hosted.

##### In ambassador.git,

- In the branch `release/v1.4`,
  - In `docs/js/doc-page.js`, update Sidebar's prefix to `/docs/latest`
  ```js
  <Sidebar location={location} prefix="/docs/latest" items={docLinks} />
  ```

##### In getambassador.io.git,
- Add the submodule pointing to `release/v1.4` branch to the `submodules/1.4/` directory
```
git submodule add --name ambassador-1.4 --branch release/v1.4 https://github.com/datawire/ambassador.git submodules/1.4/
```
- Now link only the docs in this branch under `/docs-structure/docs/` directory.
```
cd docs-structure/docs/
ln -s ../../submodules/1.4/docs 1.4
```
- Add 1.4 dropdown link in `src/components/Header/Header.js` file.
```js
              <li>
                <div className={classnames(styles.Dropdown, !isDocLink((location || {}).pathname) && styles.hidden)}>
                  <button className={classnames(styles.DropdownButton, styles.DocsDropdownColor)}>{ docsVersion((location || {}).pathname) } ▾</button>
                  <div className={styles.DropdownContent}>
                    <Link to="/docs/latest/">Latest</Link>
                    <Link to="/docs/1.4/">v1.4</Link>
                    <Link to="/docs/1.3/">v1.3</Link>
                    <Link to="/docs/1.2/">v1.2</Link>
                    <Link to="/docs/1.2/">v1.1</Link>
                  </div>
                </div>
              </li>
```

##### Note:
Now, the website must now display v1.4 docs under getambassador.io/docs/. Make sure everything looks right.

##### In ambassador.git,

Now that the latest release is `1.4`, we need to remove that tag from `1.3`.

- In the branch `release/v1.3`,
  - In `docs/js/doc-page.js`, update Sidebar's prefix to `/docs/1.3`
  ```js
  <Sidebar location={location} prefix="/docs/1.3" items={docLinks} />
  ```

##### Note:
You should be all set now.
- `/docs/latest/` must now show v1.4 docs.
- `/docs/1.4/` must now point to `/docs/latest/`.
- `/docs/1.3/` must now show docs under `release/v1.3` branch.
- The rest of the versions must also show the right set of docs.
