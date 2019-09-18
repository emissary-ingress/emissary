# How it works

Devportal allows user to publish reference doucmentation (static text) and API
documentation about services fronted by Ambassador

## Rendering

- Reference documentation is authored in Markdown and processed with Blackfriday
  processor
- page layout and navigation is done using HTML fragments that are processed as
  Go html templates
- api documentation is done using HTML fragment (using swaggerui javascript
  library)
- static files CSS and assets are served directly

### landing page

corresponds to `$prefix/`

Served from file `/landing.gomd`

### Service API pages

corresponds to `$prefix/doc/$namespace/$service`

Implemented by 

#### Service discovery

List of services is obtained from the Ambassador diag API by polling.

The swagger is rewritten slightly to account for service url prefix rewriting
done by ambassador.

### Reference documentation pages

served from `$prefix/pages/$page`

One page corresponds to the file `/pages/$page.gomd`. Currently the name of the
file is the name of the page. The list of pages is loaded from the folder
`/pages/*.gomd`

Since go templates do not support dynamic template invocations a `///page-magic`
go template is dynamically generated for the current page by the devportal to
bind layout and page together

### layout

Layout of all pages is the actual top-level go template that is rendered and it
should invoke other templates based on context variables. The layout template is
loaded from `/layout.gohtml` file.

### markup fragment templates

Templates are loaded from files in `/fragments/$template.gohtml` and are named
by their filename.

## Content source

Content source can be either a path or an URL. If it's a path it's considered to
be a local filesystem. If it's a (http[s]) url it's considered to be an URL to a
git repo. The git repo is then checked out into a memory filesystem.

# running devportal

## local development

To run only the devportal locally, run 

        # make run-dev-portal
        
This will start a mock of ambassador (`fake-ambassador.py`) and the dev portal.

Open the browser at [http://localhost:8877/]

Both processes can be gracefully stopped with `^C`

The script contains instructions how to set up dev portal content.

## testing with actual ambassador

See [README.md] for details

        # make deploy proxy
        
Open the browser at [https://ambassador.default/docs/]

# making a release

See [README.md] for details
