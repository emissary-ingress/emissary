# Rewrites

Once Ambassador Edge Stack uses a prefix to identify the service to which a given request should be passed, it can rewrite the URL before handing it off to the service. 

There are two approaches for rewriting: `rewrite` for simpler scenarios and `regex_rewrite` for more advanced rewriting.

**Please note that** only one of these two can be configured for a mapping **at the same time**. As a result Ambassador Edge Stack ignores `rewrite` when `regex_rewrite` is provided.

## `rewrite`


By default, the `prefix` is rewritten to `/`, so e.g., if we map `/prefix1/` to the service `service1`, then

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

Ambassador Edge Stack can be configured to not change the prefix as it forwards a request to the upstream service. To do that, specify an empty `rewrite` directive:

- `rewrite: ""`

## `regex_rewrite`

In some cases, a portion of URL needs to be extracted before making the upstream service URL. For example, suppose that when a request is made to `leaderboards/v1/12345/find`, the target URL must be rewritten as `game/12345`. We can do this as follows:

```shell
prefix: /leaderboards/
regex_rewrite:
    pattern: 'leaderboards/v1/([0-9]*)/find'
    substitution: '/game/\1'
```

`([0-9]*)` can be replaced with `(\d)` for simplicity.

More than one group can be captured in the `pattern` to be referenced by `\2`, `\3` and `\n` in the `substitution` section.

For more information on how rewrite and prefix can be configured, see [Mappings](../mappings).
