# Rewrites

Once Ambassador Edge Stack uses a prefix to identify the service to which a given request should be passed, it can rewrite the URL before handing it off to the service.

There are two approaches for rewriting: `rewrite` for simpler scenarios and `regex_rewrite` for more advanced rewriting.

**Please note that** only one of these two can be configured for a mapping **at the same time**. As a result Ambassador Edge Stack ignores `rewrite` when `regex_rewrite` is provided.

## `rewrite`

By default, the `prefix` is rewritten to `/`, so e.g., if we map `/backend-api/` to the service `service1`, then

<samp>http://ambassador.example.com<span style="color:red">/backend-api/</span><span style="color:green">foo/bar</span></samp>

* `prefix`: <samp style="color:red">/backend-api/</samp> which rewrites to <samp style="color:red">/</samp> by default.
* `rewrite`: <samp style="color:red">/</samp>
* `remainder`: <samp style="color:green">foo/bar</samp>


would effectively be written to

<samp>http://service1<span style="color:red">/</span><span style="color:green">foo/bar</span></samp>

* `prefix`: was <samp style="color:red">/backend-api/</samp>
* `rewrite`: <samp style="color:red">/</samp> (by default)

You can change the rewriting: for example, if you choose to rewrite the prefix as <samp style="color:red">/v1/</samp> in this example, the final target would be:


<samp>http://service1<span style="color:red">/v1/</span><span style="color:green">foo/bar</span></samp>

* `prefix`: was <samp style="color:red">/backend-api/</samp>
* `rewrite`: <samp style="color:red">/v1/</samp>

And, of course, you can choose to rewrite the prefix to the prefix itself, so that

<samp>http://ambassador.example.com<span style="color:red">/backend-api/</span><span style="color:green">foo/bar</span></samp>

* `prefix`: <samp style="color:red">/backend-api/</samp>
* `rewrite`: <samp style="color:red">/backend-api/</samp>

would be "rewritten" as:

<samp>http://service1<span style="color:red">/backend-api/</span><span style="color:green">foo/bar</span></samp>

To prevent Ambassador rewrite the matched prefix to `/` by default, it can be configured to not change the prefix as it forwards a request to the upstream service. To do that, specify an empty `rewrite` directive:

- `rewrite: ""`

In this case requests that match the prefix <samp style="color:red">/backend-api/</samp> will be forwarded to the service without any rewriting:

<samp>http://ambassador.example.com<span style="color:red">/backend-api/</span><span style="color:green">foo/bar</span></samp>

would be forwarded to:

<samp>http://service1<span style="color:red">/backend-api/</span><span style="color:green">foo/bar</span></samp>

## `regex_rewrite`

In some cases, a portion of URL needs to be extracted before making the upstream service URL. For example, suppose that when a request is made to `foo/12345/list`, the target URL must be rewritten as `/bar/12345`. We can do this as follows:

```shell
prefix: /foo/
regex_rewrite:
    pattern: '/foo/([0-9]*)/list'
    substitution: '/bar/\1'
```

`([0-9]*)` can be replaced with `(\d)` for simplicity.

<samp>http://ambassador.example.com<span style="color:red">/foo/</span><span style="color:green">12345/list</span></samp>

* `prefix`: <samp style="color:red">/foo/</samp>
* `pattern`: <samp style="color:green">/foo/<span style="color:DarkSlateBlue">12345</span>/list</samp> where <samp style="color:DarkSlateBlue">12345</samp> is captured by `([0-9]*)`
* `substitution`:  <samp style="color:brown">/bar/<span style="color:DarkSlateBlue">12345</span></samp> where <samp style="color:DarkSlateBlue">12345</samp> is substituted by `\1`

would be forwarded to:

<samp>http://service1<span style="color:brown">/bar/</span><span style="color:DarkSlateBlue">12345</span></samp>

More than one group can be captured in the `pattern` to be referenced by `\2`, `\3` and `\n` in the `substitution` section.

For more information on how `Mapping` can be configured, see [Mappings](../mappings).
