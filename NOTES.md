
# Gotchas

- You can't use localhost to configure ambassador to talk to the
  sidecar, this resolves to an ipv6 address ([::]) and for some reason
  that doesn't work.

- The ratelimit service only sees files that start with "config.", so
  you need to name all the configuration files accordingly.

- There can be only one config file per domain.

# Issues

- Ambassador hardcodes the domain to "ambassador"

- Ambassador doesn't give much control over how descriptors are
  created. It always includes all the rate limit actions in a
  hardcoded order. This makes writing rate limit rules more
  complicated and also more limited.

# Todo

- Define a custom CRD

- Wire up a k8s watcher

- Figure out how to restart the sidecar
