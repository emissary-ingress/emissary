# Rewrites

Once Ambassador Edge Stack uses a prefix to identify the service to which a given request should be passed, it can rewrite the URL before handing it off to the service. 

There are two approaches for rewriting: `rewrite` for simpler scenarios and `regex_rewrite` for more advanced rewriting.

**Please note that** only one of these two can be configured for a mapping **at the same time**. As a result Ambassador Edge Stack ignores `rewrite` when `regex_rewrite` is provided.

## `rewrite`

By default, the `prefix` is rewritten to `/`, so e.g., if we map `/backend-api/` to the service `service1`, then

<code>
http://ambassador.example.com<span style="color:red">/backend-api/</span><span style="color:green">foo/bar</span>
</code>

* ```prefix```: <span style="color:red">/backend-api/</span> which rewrites to <span style="color:red">/</span> by default.
* ```rewrite```: <span style="color:red">/</span>
* ```remainder```: <span style="color:green">foo/bar</span>


would effectively be written to

<code>
http://service1<span style="color:red">/</span><span style="color:green">foo/bar</span>
</code>

* ```prefix```: was <span style="color:red">/backend-api/</span>
* ```rewrite```: <span style="color:red">/</span> (by default)

You can change the rewriting: for example, if you choose to rewrite the prefix as <span style="color:red">/v1/</span> in this example, the final target would be:


<code>
http://service1<span style="color:red">/v1/</span><span style="color:green">foo/bar</span>
</code>

* ```prefix```: was <span style="color:red">/backend-api/</span> 
* ```rewrite```: <span style="color:red">/v1/</span>

And, of course, you can choose to rewrite the prefix to the prefix itself, so that

<code>
http://ambassador.example.com<span style="color:red">/backend-api/</span><span style="color:green">foo/bar</span>
</code>

* ```prefix```: <span style="color:red">/backend-api/</span>
* ```rewrite```: <span style="color:red">/backend-api/</span>

would be "rewritten" as:

<code>
http://service1<span style="color:red">/backend-api/</span><span style="color:green">foo/bar</span>
</code>

Ambassador can be configured to not change the prefix as it forwards a request to the upstream service. To do that, specify an empty `rewrite` directive:

- `rewrite: ""`

## `regex_rewrite`

In some cases, a portion of URL needs to be extracted before making the upstream service URL. For example, suppose that when a request is made to `foo/12345/list`, the target URL must be rewritten as `bar/12345`. We can do this as follows:

```shell
prefix: /foo/
regex_rewrite:
    pattern: '/foo/([0-9]*)/list'
    substitution: '/bar/\1'
```

`([0-9]*)` can be replaced with `(\d)` for simplicity.

More than one group can be captured in the `pattern` to be referenced by `\2`, `\3` and `\n` in the `substitution` section.

For more information on how `Mapping` can be configured, see [Mappings](../mappings).
