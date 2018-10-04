# Five minute quickstart

In this section, we'll get Ambassador running locally with a demo configuration. In the next section, we'll then walk through how to deploy Ambassador in Kubernetes with a custom configuration.

## 1. Running the Demo Configuration

By default, Ambassador uses a demo configuration to show some of its basic features. Get it running with Docker, and expose Ambassador on port 8080:

```shell
docker run -it -p 8080:80 --name=ambassador --rm quay.io/datawire/ambassador:{VERSION} --demo
```

## 2. Ambassador's Diagnostics

Ambassador provides live diagnostics viewable with a web browser. While this would normally not be exposed to the public network, the Docker demo publishes the diagnostics service at the following URL:

`http://localhost:8080/ambassador/v0/diag/`

Some of the most important information - your Ambassador version, how recently Ambassador's configuration was updated, and how recently Envoy last reported status to Ambassador - is right at the top. The diagnostics overview can show you what it sees in your configuration map, and which Envoy objects were created based on your configuration.

## 3. The Quote of the Moment Service

Since Ambassador is an API gateway, its primary purpose is to provide access to microservices. The demo is preconfigured with a mapping that connects the `/qotm/` resource to the "Quote of the Moment" service -- a demo service that supplies quotations. You can try it out here:

```shell
curl http://localhost:8080/qotm/
```

This request will route to the `qotm` service at `demo.getambassador.io`, and return a quote in a JSON object.

You can also see the mapping by clicking the `mapping-qotm.yaml` link from the diagnostic overview, or by opening

`http://localhost:8080/ambassador/v0/diag/mapping-qotm.yaml`

## 4. Authentication

On the diagnostic overview, you can also see that Ambassador is configured to do authentication -- click the `auth.yaml` link, or open

`http://localhost:8080/ambassador/v0/diag/auth.yaml`

for more here. Ambassador uses a demo authentication service at `demo.getambassador.io` to mediate access to the Quote of the Moment: simply getting a random quote is allowed without authentication, but to get a specific quote, you'll have to authenticate:

```shell
curl -v http://localhost:8080/qotm/quote/5
```

will return a 401, but

```shell
curl -v -u username:password http://localhost:8080/qotm/quote/5
```

will succeed. (Note that that's literally "username" and "password" -- the demo auth service is deliberately not very secure!)

Note that it's up to the auth service to decide what needs authentication -- teaming Ambassador with an authentication service can be as flexible or strict as you need it to be.

## Next steps

We've just walked through some of the core features of Ambassador in a local configuration. Next, we'll walk through how to configure these features in Kubernetes.
