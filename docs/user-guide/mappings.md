---
layout: doc
weight: 2
title: "About Mappings"
categories: user-guide
---

At the heart of Ambassador is the idea of _mappings_, which associate _resources_ (in the REST sense) to _services_ (in the Kubernetes sense), applying a _rewrite rule_ in the process.

### Mappings

You use `POST` requests to the admin interface to map a resource to a service:

```
curl -XPOST -H "Content-Type: application/json" \
      -d '{ "prefix": "<url-prefix>", "service": "<service-name>", "rewrite": "<rewrite-as>" }' \
      http://localhost:8888/ambassador/mapping/<mapping-name>
```

where

- `<mapping-name>` is a unique name that identifies this mapping
- `<url-prefix>` is the URL prefix identifying your [resource](#resources)
- `<service-name>` is the name of the [service](#services) handling the resource
- `<rewrite-as>` is what to [replace](#rewriting) the URL prefix with when talking to the service

You can get a list of all the mappings that Ambassador knows about with

```
curl http://localhost:8888/ambassador/mappings
```

and you can delete a specific mapping with

```
curl -XDELETE http://localhost:8888/ambassador/mapping/<mapping-name>
```

Also, the `mapping-name` identifies the mapping in statistics output and such.

## <a name="resources">Resources</a>

To Ambassador, a `resource` is a group of one or more URLs that all share a common prefix in the URL path. For example:

```
https://ambassador.example.com/resource1/foo
https://ambassador.example.com/resource1/bar
https://ambassador.example.com/resource1/baz/zing
https://ambassador.example.com/resource1/baz/zung
```

all share the `/resource1/` path prefix, so can be considered a single resource. On the other hand:

```
https://ambassador.example.com/resource1/foo
https://ambassador.example.com/resource2/bar
https://ambassador.example.com/resource3/baz/zing
https://ambassador.example.com/resource4/baz/zung
```

share only the prefix `/` -- you _could_ tell Ambassador to treat them as a single resource, but it's probably not terribly useful.

Note that the length of the prefix doesn't matter: if you want to use prefixes like `/v1/this/is/my/very/long/resource/name/`, go right ahead, Ambassador can handle it.

Also note that Ambassador does not actually require the prefix to start and end with `/` -- however, in practice, it's a good idea. Specifying a prefix of

```
/man
```

would match all of the following:

```
https://ambassador.example.com/man/foo
https://ambassador.example.com/mankind
https://ambassador.example.com/man-it-is/really-hot-today
https://ambassador.example.com/manohmanohman
```

which is probably not what was intended.

## <a name="services">Services</a>

A `service` is exactly the same thing to Ambassador as it is to Kubernetes. When you tell Ambassador to map a resource to a service, it requires there to be a Kubernetes service with _exactly_ the same name, and it trusts whatever the Kubernetes has to say about ports at such.

At present, Ambassador relies on Kubernetes to do load balancing: it trusts that using the DNS to look up the service by name will do the right thing in terms of spreading the load across all instances of the service. This will change shortly, in order to gain better control of load balancing.

## <a name="rewriting">Rewrite Rules</a>

Once Ambassador uses a prefix to identify the service to which a given request should be passed, it can rewrite the URL before handing it off to the service. By default, the `prefix` is rewritten to `/`, so e.g. if we map `/prefix1/` to the service `service1`, then

```
http://ambassador.example.com/prefix1/foo/bar
```

would effectively be written to

```
http://service1/foo/bar
```

when it was handed to `service1`.

You can change the rewriting: for example, if you choose to rewrite the prefix as `/v1/` in this example, the final target would be

```
http://service1/v1/foo/bar
```

And, of course, you can choose to rewrite the prefix to the prefix itself, so that

```
http://ambassador.example.com/prefix1/foo/bar
```

would be "rewritten" as

```
http://service1/prefix1/foo/bar
```
