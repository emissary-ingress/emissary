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

### If you made changes to the `apictl-key serve` command, then:

1. To deploy a new aes backend, run:

```
PROD_KUBECONFIG=<prod-kubeconfig-file> make deploy-aes-backend
```

