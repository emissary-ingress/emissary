## EDGE STACK RELEASE PROCESS

You can only do RC releases so far.

### THE CI WAY

1.

```
make aes-rc
```

### THE MANUAL WAY

Instead of above, you can do:

1.

```
make aes-rc-now
```

### If you made changes to the `apictl-key serve-aes-backend` command, then:

1. To build and push a new aes backend, run:

```
make aes-backend-push AES_BACKEND_RELEASE_REGISTRY=gcr.io/datawireio AES_BACKEND_RELEASE_VERSION=$RELEASE_VERSION
```

