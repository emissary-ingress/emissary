# Releases

This document describes the release process for Emissary. It is intended
for maintainers cutting a release. Contributors should not need to read
it.

The examples below use `v4.1` and `v4.1.0-rc.0`. Substitute the version
you are actually cutting. All releases follow the same process, whether
they're an RC, major, minor, or patch release.

## Release shepherd responsibilities

One maintainer takes the role of release shepherd for a given minor
release. The shepherd is responsible for:

- Cutting the release branch and the initial release candidate.
- Triaging bug reports against the release branch, deciding what gets
  backported, and cutting any follow-up RCs.
- Cutting the final `vX.Y.0` once the RCs are stable.
- Cutting patch releases (`vX.Y.Z`) off the release branch as needed.

## Branching and cherry-pick policy

Each release lives on its own long-lived branch named `release/vX.Y` (for
example `release/v4.1`). Tags for that minor (RCs, the GA `vX.Y.0`
release, and any subsequent `vX.Y.Z` patches) are cut from this branch.

The flow:

- New features and other non-bugfix changes land on `main`.
- Bugfixes intended for an active release land on `release/vX.Y` and are
  cherry-picked back to `main`, unless there's a specific reason to do
  the reverse.

Going branch-first (instead of main-first) for bugfixes helps keep the
release branch independently buildable and avoids losing fixes if a
feature commit on `main` isn't safe to backport.

## How to cut a release

### 1. Cut the release branch (new major or minor only)

If `release/vX.Y` doesn't already exist, you'll need to create it from
`main` at the point you are ready to cut `vX.Y.0-rc.0`:

```bash
git fetch origin
git checkout -b release/v4.1 origin/main
git push origin release/v4.1
```

For a patch release on an existing minor, the branch MUST already exist.
If you think you need to do a patch and the release branch doesn't
already exist, **STOP** and consult with the other maintainers.

### 2. Make any release-branch-only changes

Release-prep tweaks happen on `release/vX.Y`, not on `main`. The first
step of any release is to make sure that the release branch has all the
commits that should be part of the release.

### 3. Prepare the CHANGELOG

Changelog entries live as snippets in `docs/changes/`, as described in
`docs/changes/README.md`. Those need to be assembled into the main
`CHANGELOG.md` file:

```bash
make release/changelog-update CHANGELOG_VERSION=4.1.0-rc.0   # or 4.1.0
```

`changelog-update` will replace all the entries in `CHANGELOG` since the
previous GA: this is expected. If something goes wrong (or if all the snippets are `skip`s), it will fail. **Fix it before continuing**.

Assuming that `changelog-update` completes, review its work. If anything
looks off, **edit the snippets and rerun `make release/changelog-update`** --
edits you make by hand to `CHANGELOG.md` will not be preserved.

Once you're satisfied with `CHANGELOG.md`, commit it along with any
snippet edits (the tag must contain the snippets that built the notes):

```bash
git add CHANGELOG.md docs/changes
git commit -m "docs(changelog): cut 4.1.0-rc.0 release notes"
```

Land this on `release/v4.1` (via a PR) before tagging.

**Note**: `changelog-update` does not remove the snippets: they stay on
disk until explicitly removed (see step 7) so that you can edit them if
`CHANGELOG.md` doesn't look right.

### 4. Tag the release

Tag the tip of the release branch:

```bash
git checkout release/v4.1
git pull origin release/v4.1
git tag -s -m "v4.1.0-rc.0" v4.1.0-rc.0
git push origin v4.1.0-rc.0
```

The same procedure is used for RCs (`v4.1.0-rc.N`) and GA releases.

### 5. Wait for CI

Pushing the tag triggers the full CI pipeline. Lint and tests run first;
the release job is gated on them and will not run if any of them fail.
When the release job does run, it pushes images and Helm charts to GHCR
(under `ghcr.io/emissary-ingress`) and creates a corresponding GitHub
Release.

Watch the run, and if it fails, fix forward. Do not delete and re-push the
tag.

### 6. Publish the GitHub Release

The release is created as a **draft** and is **not** marked as the latest
release. Once CI is green:

- Open the draft release on GitHub and review the body.
- Manually edit the release notes to include the relevant changes from
  the CHANGELOG.
- Publish the release.
- For a GA or patch release (not an RC), mark it as the latest release.

### 7. Non-RC only: Clean up snippets

If you are _not_ doing an RC, clean up the snippets that went into the
release:

```bash
make release/changelog-consume
```

This will clean up all the snippets used in the release just tagged. If
you have modified any of those snippets, it will fail: you need to rename
the snippet or delete it by hand. And, again, **do not** do this for RCs.
Check to make sure things look sane, then commit:

```bash
git add ... # hopefully just CHANGELOG and snippets
git commit -m "docs(changelog): clean up snippets released in 4.1.0"
```

**Note**: CI fails if you forget this step, so don't.

### 8. Cherry-pick back

Cherry-pick appropriate changes back to `main`. This _should_ just
involve `CHANGELOG.md` and snippets.

**Note**: CI fails if you forget this step, so don't.
