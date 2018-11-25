# Rewrites

Once Ambassador uses a prefix to identify the service to which a given request should be passed, it can rewrite the URL before handing it off to the service. By default, the `prefix` is rewritten to `/`, so e.g., if we map `/prefix1/` to the service `service1`, then

```shell
http://ambassador.example.com/prefix1/foo/bar
```

would effectively be written to

```shell
http://service1/foo/bar
```

when it was handed to `service1`.

You can change the rewriting: for example, if you choose to rewrite the prefix as `/v1/` in this example, the final target would be:

```shell
http://service1/v1/foo/bar
```

And, of course, you can choose to rewrite the prefix to the prefix itself, so that

```shell
http://ambassador.example.com/prefix1/foo/bar
```

would be "rewritten" as:

```shell
http://service1/prefix1/foo/bar
```

Ambassador can be configured to not change the prefix as it forwards a request to the upstream service. To do that, specify an empty `rewrite` directive:

- `rewrite: ""`

