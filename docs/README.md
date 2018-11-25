# Ambassador documentation

We've switched to GatsbyJS for generating the documentation, which gives us more control and flexibility over the layout.

## Getting Started
- Clone this repository
- Run `npm install` to install all dependencies
- Run `npm start` to start the development server

## Authoring documentation
To add pages add a new markdown file to the `/content` directory. Any files with a `.md` extension in the
`/content` directory will be turned into a html page at build time. To add the page to the docs sidebar, add a line to
the `/content/doc-links.yml`.

## Documentation infrastructure notes

* The rendered YAML and markdown files are copied by Travis CI to a separate Gatsby-based toolchain.
* The `doc-links.yml` file is the new TOC.