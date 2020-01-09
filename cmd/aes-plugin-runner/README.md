# aes-plugin-runner: Run Ambassador Edge Stack Filter plugins locally

`aes-plugin-runner` lets you run an Ambassador Edge Stack Filter
plugin as a stand-alone Ambassador AuthService, making it much easier
to develop the plugin.

## Usage:

	$ aes-plugin-runner --help
	Usage: aes-plugin-runner TCP_ADDR PATH/TO/PLUGIN.so
	   or: aes-plugin-runner <-h|--help>
	Run an Ambassador Pro filter as an Ambassador AuthService, for plugin development
	

You can then use `curl` to create an HTTP request, and examine the subsequent response.

## Example:

	$ aes-plugin-runner :8080 ./wiki-plugin.so
	...
	$ curl -v localhost:8080
	* Rebuilt URL to: localhost:8080/
	*   Trying ::1...
	* TCP_NODELAY set
	* Connected to localhost (::1) port 8080 (#0)
	> GET / HTTP/1.1
	> Host: localhost:8080
	> User-Agent: curl/7.54.0
	> Accept: */*
	>
	< HTTP/1.1 200 OK
	< X-Wikipedia: https://en.wikipedia.org/wiki/Long_Street,_Buckinghamshire
	< Date: Mon, 25 Feb 2019 20:21:35 GMT
	< Content-Length: 0
	<
	* Connection #0 to host localhost left intact
