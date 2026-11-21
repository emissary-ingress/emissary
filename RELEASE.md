# Releases

This document describes the release process for Emissary. It is intended for
maintainers cutting a release. Contributors should not need to read it.

The examples below use `v4.1` and `v4.1.0-rc.0`. Substitute the version you are
actually cutting.

## Release shepherd responsibilities

One maintainer takes the role of release shepherd for a given minor release.
The shepherd is responsible for:

- Cutting the release branch and the initial release candidate.
- Triaging bug reports against the release branch, deciding what gets
  backported, and cutting any follow-up RCs.
- Cutting the final `vX.Y.0` once the RCs are stable.
- Cutting patch releases (`vX.Y.Z`) off the release branch as needed.

## Branching and cherry-pick policy

Each minor release lives on its own long-lived branch named `release/vX.Y`
(for example `release/v4.1`). Tags for that minor (RCs, the GA release, and
all subsequent patches) are cut from this branch.

The flow:

- New features and other non-bugfix changes land on `main`.
- Bugfixes intended for an active release land on `release/vX.Y` and are
  cherry-picked back to `main`, unless there's a specific reason to do the
  reverse.

Going branch-first (instead of main-first) for bugfixes keeps the release
branch independently buildable and avoids losing fixes if a feature commit on
`main` isn't safe to backport.

## How to cut a release

### 1. Cut the release branch (new minor only)

For a new minor release, create the release branch from `main` at the point
you are ready to cut `vX.Y.0-rc.0`:

```
git fetch origin
git checkout -b release/v4.1 origin/main
git push origin release/v4.1
```

For a patch release on an existing minor, skip this step as the branch
already exists.

### 2. Make any release-branch-only changes

Release-prep tweaks happen on `release/vX.Y`, not on `main`.

Changelog entries live as snippets in `docs/changes/` (see its README). Before
tagging any release (RC, GA, or patch), assemble them into `CHANGELOG.md` on the
release branch and commit the result:

```
make release/changelog-update CHANGELOG_VERSION=4.1.0-rc.0   # or 4.1.0
git add -A CHANGELOG.md docs/changes
git commit -m "docs(changelog): cut 4.1.0-rc.0 release notes"
```

This rebuilds the whole section for that version below the `towncrier release
notes start` marker, replacing any earlier `4.1.0-rc.N` sections:

- For an **RC**, the snippets are kept, so the next RC (or the GA) is rebuilt
  from everything that has landed so far.
- For a **GA or patch** release, the snippets are consumed (deleted), and only
  the GA section remains.

Use `uv run --dev -- towncrier build --draft --version 4.1.0` to preview
without writing anything.

The GitHub Release body links to `CHANGELOG.md` at the tag (it does not
inline the changelog), so make sure the section for the version you're
about to tag is in good shape on the release branch before tagging.

After a GA or patch release, cherry-pick the changelog commit back to `main` so
the consumed snippets are removed there too. Don't do this for RCs.

If you do make changes, cherry-pick them back to `main` afterwards per the
policy above.

### 3. Tag the release

Tag the tip of the release branch:

```
git checkout release/v4.1
git pull origin release/v4.1
git tag -s -m "v4.1.0-rc.0" v4.1.0-rc.0
git push origin v4.1.0-rc.0
```

The same procedure is used for RCs (`v4.1.0-rc.N`), the GA release
(`v4.1.0`), and patch releases (`v4.1.Z`).

### 4. Wait for CI

Pushing the tag triggers the full CI pipeline. Lint and tests run first; the
release job is gated on them and will not run if any of them fail. When the
release job does run, it pushes images and Helm charts to GHCR (under
`ghcr.io/emissary-ingress`) and creates a corresponding GitHub Release.

Watch the run, and if it fails, fix forward. Do not delete and re-push the
tag.

### 5. Publish the GitHub Release

The release is created as a **draft** and is **not** marked as the latest
release. Once CI is green:

- Open the draft release on GitHub and review the body.
- Manually edit the release notes to include the relevant changes from the CHANGELOG.
- Publish the release.
- For a GA or patch release (not an RC), mark it as the latest release.

## Patch releases

Patch releases are cut from the same `release/vX.Y` branch:

1. Land the bugfix on `release/vX.Y` (cherry-picked from `main` or authored
   directly on the branch).
2. Cherry-pick to `main` if the fix is also relevant there.
3. Tag `release/vX.Y` with `vX.Y.Z` and push the tag.
