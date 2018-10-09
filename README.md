# Ambassador documentation

We've switched to GatsbyJS for generating the documentation, which gives us more control and flexibility over the layout.

## Authoring documentation

If you're authoring the documentation, just edit the Markdown files. You can use GitHub to preview the Markdown.

## Documentation infrastructure notes

* The rendered YAML and markdown files are copied by Travis CI to a separate Gatsby-based toolchain. Still TODO is to provide a local version of this toolchain.
* The `doc-links.yml` file is the new TOC.