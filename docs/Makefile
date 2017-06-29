# Serve the book off a webserver:
.PHONY: serve
serve:
	node_modules/.bin/gitbook serve

# Build the book:
.PHONY: build
build:
	node_modules/.bin/gitbook build

# Install necessary software:
.PHONY: setup
setup: node_modules
	npm install gitbook-cli

node_modules:
	mkdir node_modules
