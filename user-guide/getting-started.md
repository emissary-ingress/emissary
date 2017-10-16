# Getting Started with Ambassador

Ambassador is an API Gateway for microservices. We'll demo the very basics here.

## 1. Running the Demo

You can use `docker run` to start Ambassador with a default configuration that can route to some Datawire cloud services as a demonstration:

```shell
docker run -it -p 8080:80 --name=ambassador --rm datawire/ambassador:{VERSION}
```

will start an Ambassador running in Docker that you can talk to on `localhost` port 8080.

## 2. Ambassador's Diagnostics

Ambassador provides live diagnostics viewable with a web browser. While this would normally not be exposed to the public network, the Docker demo makes the diagnostics available at

`http://localhost:8080/ambassador/v0/diag/`

Some of the most important information - your Ambassador version, how recently Ambassador's configuration was updated, and how recently Envoy last reported status to Ambassador - is right at the top. The diagnostics overview can show you what it sees in your configuration map, and which Envoy objects were created based on your configuration.

## 3. The Quote of the Moment service

Since Ambassador is an API gateway, its primary purpose is to provide access to microservices. The demo is preconfigured with a mapping that connects the `/qotm/` resource to the "Quote of the Moment" service -- a demo service that supplies (usually surreal) quotations.

You can see the mapping by clicking the `mapping-qotm.yaml` link from the diagnostic overview, or by opening

`http://localhost:8080/ambassador/v0/diag/mapping-qotm.yaml`

To _use_ the mapping to get a Quote of the Moment, just

```shell
curl http://localhost:8080/qotm/
```

This will be routed to the `qotm` service at `demo.getambassador.io`, and should return something like

```json
{
  "hostname": "qotm-1827164760-180sg",
  "ok": true,
  "quote": "Utter nonsense is a storyteller without equal.",
  "time": "2017-10-16T18:53:13.265612",
  "version": "1.1"
}
```

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

## 5. Using Your Own Config

If you want to run Ambasasdor with a custom configuration, it's as simple as assembling a directory full of YAML files and using a two-line Dockerfile:

```shell
FROM datawire/ambassador:{VERSION}
COPY config /etc/ambassador-config
```

You can also use a `ConfigMap` to allow configuration updates without building a new image: Ambassador is designed for flexibility and self-service.

That's the basics. For more:

- Join us on [Gitter](https://gitter.im/datawire/ambassador);
- Learn how to [add authentication](auth-tutorial.md) to existing services; or
- Dig into more about [running Ambassador](running.md).
