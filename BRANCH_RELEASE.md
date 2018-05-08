# Branch and Release Model

The branch and release model for this project is a straightforward trunk-based development flow where the project tries to keep `master` in working condition as often as possible and developers use branches from the trunk to make changes and pull requests to merge the code back into the trunk. 

# Development Changes

## Development

- The `master` branch is the "trunk" and the project maintainers will make a good faith effort to keep it stable and resolve instability problems in a reasonable timeframe.
- Changes occur on branches.
- There is a special type of branch for minor or "low impact" changes called `nobuild`. Any branch that matches `^nobuild.*` skips all CI activities.
- When a change on a branch is at a point where there change is ready for review and testing then a developer should open a GitHub PR that targets the `master` branch with the code on their branch.
    - When a PR is started or changed the GitHub status check system will ensure CI performs a new build and runs the entire e2e suite.
- Maintainers merge branches into `master` after review and testing passes. When a branch is merged into `master` then CI once again builds and runs the entire e2e suite.
- Mainters delete the merged branch.

## Release

Before beginning this process determine the version number!

- Releases are trigged from tags.
- There should always be at least one release candidate ("RC") before a general availability ("GA") stable release as the release candidate build and testing process is when human-friendly version information is "burned" into the release artifact.
- Release candidate tags should be named `-rc${Unsigned-Integer}` and should be successive, for example: `-rc0`, `-rc1`, `-rc2`.
- CI will perform a full build and e2e test for a release release candidate.
    - If the release candidate fails a Maintainer should investigate and fix the issue. Using the normal Development Workflow is ideal, but can be circumvented if necessary.
- Once CI has completed and tests pass then a GA release tag can be pushed. A GA release tag is the version used in the release candidate but without the `-rc` suffix.
- A GA release tag **DOES NOT** cause the artifacts to be rebuilt or tested again. The previously built Docker tags are pulled from the Docker repository (using the Git SHA of the commit pointed at by the tag). The Docker images are tagged with the final GA version number and then pushed again.

# Documentation Change

## Minor Change

If the change is minor, for example, to fix punctuation, grammar, or a broken link then follow the below instructions:

1. Create a new `nobuild` branch from `master`: `git checkout -b nobuild/${change-name}`
2. Modify the documentation as necessary.
3. Commit the changed files to your local branch `git add ...` and then `git commit -m '${short-descriptive-message}'`.
4. Push the change to the Git remote `git push origin nobuild/${change-name}`.
5. Submit a GitHub PR for your branch that targets `master`.

## Major Change

If the change is major, for example, to redesign the site then follow the below instructions:

1. Create a new `nobuild` branch from `master`: `git checkout -b doc/${change-name}`
2. Modify the documentation as necessary.
3. Commit the changed files to your local branch `git add ...` and then `git commit -m '${short-descriptive-message}'`.
4. Push the change to the Git remote `git push origin doc/${change-name}`.
5. Submit a GitHub PR for your branch that targets `master`.
6. CI will pickup your change and build the website then it will publish a draft to Netlify.
