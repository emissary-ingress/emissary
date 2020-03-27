# Ambassador documentation

We've switched to GatsbyJS for generating the documentation, which gives us more control and flexibility over the layout.

## Authoring documentation

 - If you are improving the documentation of for the current release
   of Ambassador or Ambassador Edge Stack, you should submit your
   pull-request to [ambassador-docs.git][].
 - If you are writing documentation for changes in an upcoming release
   of Ambassador, you should submit your pull-request to
   [ambassador.git][] (in the `docs/` folder).
 - If you are writing documentation for changes in an upcoming release
   of Ambassador Edge Stack, you should submit your pull-request to
   [apro.git][] (in the `docs/` folder).


If you're authoring the documentation, just edit the Markdown files. You can use GitHub to preview the Markdown.

In both YAML and Markdown files, strings like `$variable$` are
substituted with the values defined in `versions.yml`.

The `doc-links.yml` file is the table-of-contents.

The `pro-pages.yml` file identifies which pages should be marked as
"Pro" pages.

## Documentation infrastructure notes

The docs canonically live at [ambassador.git][] under `/docs/` directory.

Pushing to the release branch of [ambassador.git][] causes
Travis CI to update [getambassador.io.git][]'s subtree of the docs,
which will cause a website update.  That repository contains the
Gatsby-based toolchain that compiles the docs in to a website. Still
TODO is to provide a local/public version of this toolchain.

Repositories that include the docs as a subtree should get in the
habit of doing a `git subtree pull` from their `master` branch
periodically.  Documentation for code changes can then be committed
right along-side the code changes.  When a release is cut, and you are
ready to publicize it, simply do a `git subtree push`.

[ambassador.git]: https://github.com/datawire/ambassador
[apro.git]: https://github.com/datawire/apro
[getambassador.io.git]: https://github.com/datawire/getambassador.io
