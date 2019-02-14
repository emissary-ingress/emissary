# apro-plugin-runner: Run Ambassador Pro middleware plugins locally

`apro-plugin-runner` lets you run an Ambassador Pro middleware plugin
as a stand-alone Ambassador AuthService, making it much easier to
develop the middleware plugin.

## Installation

	$ go get github.com/datawire/apro-plugin-runner

Then make sure `$(go env GOPATH)/bin` is in your `$PATH`.

## Usage:

	$ apro-plugin-runner --help
	Usage: apro-plugin-runner TCP_ADDR PATH/TO/PLUGIN.so
	   or: apro-plugin-runner <-h|--help>
	Run an Ambassador Pro middleware plugin as an Ambassador AuthService, for plugin development
	
	Example:
	    apro-plugin-runner :8080 ./myplugin.so
