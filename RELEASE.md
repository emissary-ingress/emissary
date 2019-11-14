## EDGE STACK RELEASE PROCESS

You can only do RC releases so far.

### THE CI WAY

1.

```
make aes-rc
```

2. Check CircleCI status at https://circleci.com/gh/datawire/apro.

3. Wait for CircleCI to show green for your release build before continuing. If the CI build fails,
   figure out why, fix it, and go back to step 1.

### THE MANUAL WAY

Instead of steps 1 -3 above, you can do:

1.

```
make aes-rc-now
```

### If you made changes to the `apictl-key serve` command, then:

1. To deploy a new aes backend, run:

```
PROD_KUBECONFIG=<prod-kubeconfig-file> make deploy-aes-backend
```

