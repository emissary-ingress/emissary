# Ambassador

Just cloned? Set up GitBook to get started.

    npm install

Now you can build the site, e.g., to push to a web server.

    npm run build

Still writing and editing? Run the server so you can preview as you go.

    npm start

Seeing this page in GitBook? We want this page to be our custom index.html page, not GitBook's generated page. At some point we must figure out how to do that.

## Ambassador features that need documentation

- Self-service
- Routing path -> service in Kubernetes
- URL rewriting
- Works with Istio
- TLS termination
- Authentication with TLS client certificates
- Stats via Ambassador
- Stats via StatsD
- Authentication with HTTP Basic Auth
- Consumers
- Authentication with an external auth service
- gRPC module for HTTP/2-only support

## Envoy features that need documentation

These are Envoy features that are made accessible via Ambassador and so ought to be at least mentioned if not documented. We should fill this in...
