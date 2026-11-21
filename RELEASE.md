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

Release-prep tweaks happen on `release/vX.Y`, not on `main`.

Changelog entries live as snippets in `docs/changes/` (see its README).
Before tagging any release (RC, GA, or patch), assemble them into
`CHANGELOG.md` on the release branch and commit the result:

```bash
make release/changelog-update CHANGELOG_VERSION=4.1.0-rc.0   # or 4.1.0
git add -A CHANGELOG.md docs/changes
git commit -m "docs(changelog): cut 4.1.0-rc.0 release notes"
```

This will rebuild the CHANGELOG: everything after the `towncrier release
notes start` marker with the same major, minor, and patch numbers will be
removed and replaced with the current content: the changelog always
includes changes since the previous **GA** version.

- For an **RC**, the snippets are kept, so the next RC (or the GA) is
  rebuilt from everything that has landed so far.
- For a **GA** release, **the snippets are consumed (deleted)** so that
  we don't have an endlessly-growing set of snippets and, importantly, so
  that the snippets on disk always reflect changes that have yet to be
  released.

Make sure that the CHANGELOG looks good before you continue. If you
really need to you can edit the CHANGELOG by hand at this one point, but
since your changes will be undone by the next `changelog-update` run, you
should instead edit snippets if necessary.

### 3. Tag the release

Tag the tip of the release branch:

```bash
git checkout release/v4.1
git pull origin release/v4.1
git tag -s -m "v4.1.0-rc.0" v4.1.0-rc.0
git push origin v4.1.0-rc.0
```

The same procedure is used for RCs (`v4.1.0-rc.N`) and GA releases.

### 4. Wait for CI

Pushing the tag triggers the full CI pipeline. Lint and tests run first;
the release job is gated on them and will not run if any of them fail.
When the release job does run, it pushes images and Helm charts to GHCR
(under `ghcr.io/emissary-ingress`) and creates a corresponding GitHub
Release.

Watch the run, and if it fails, fix forward. Do not delete and re-push the
tag.

### 5. Publish the GitHub Release

The release is created as a **draft** and is **not** marked as the latest
release. Once CI is green:

- Open the draft release on GitHub and review the body.
- Manually edit the release notes to include the relevant changes from
  the CHANGELOG.
- Publish the release.
- For a GA or patch release (not an RC), mark it as the latest release.

### 6. Cherry-pick back

Cherry-pick appropriate changes back to `main` -- usually this will just
be snippet and CHANGELOG changes, so that the `main` CHANGELOG is
consistent.
