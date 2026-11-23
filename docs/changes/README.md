# CHANGELOG snippets

Every user-visible change MUST include a file here. At release time,
[towncrier](https://towncrier.readthedocs.io/) will assemble these files
into the CHANGELOG for the release.

**DO NOT EDIT `CHANGELOG.md` directly.** Ever.

## Naming

Name your file `<issue-number>.<type>.md`, e.g. 5732.fix.md. If there's
no issue... well, first, why is there no issue? You should probably go
file one. But if you're convinced that there shouldn't be, name it
`+<short-slug>.<type>.md`.

You can include multiple snippet files for a single issue by using files
named `<issue-number>.<type>.1.md`, `<issue-number>.<type>.2.md`, etc.
However, this may be a sign that your issue should be split, but it can
be done, so think before you act!

The `<type>` field needs to line up with conventional commits: use one of
`feature`, `fix`, `update`, `change`, `deprecation`, `security`, or `skip`
(see CI, below).

## Content

**Be concise:** in most cases, you should be aiming for a single
paragraph. Write for Emissary's users, not other developers -- if a
change is only visible to developers, it probably doesn't need a
CHANGELOG snippet. Anything visible to end users (new features, Envoy
changes, breaking changes, bugfixes, deprecations, etc.) definitely DOES
need a CHANGELOG snippet.

Don't include the issue number or the type prefix; both will be
automatically added.

## CI

**PRs with no changelog snippet will fail CI.** If you're really really
certain that your change shouldn't be called out in the CHANGELOG, use a
`skip` snippet, e.g.

`+ci-tweak.skip.md`

but still put in a description of what's up and why it doesn't need to be
mentioned.

CI checks on commit and nightly to make sure that the CHANGELOG hasn't
been edited by hand, and that snippets already incorporated in the
CHANGELOG aren't still on disk.

## Previewing

You can get a preview of what towncrier will build with

```bash
uv run --dev -- towncrier build --draft --version X.Y.Z
```
