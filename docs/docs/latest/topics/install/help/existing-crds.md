# edgectl install: CRD's Already Exist
 
## The Problem

The installer found that there were existing CRD's that are incompatible with 
the version of AES that is being installed.  Unfortunately the installer does not support
upgrades or downgrades at this time.

## How to Resolve It

You can manually remove installed CRDs if you are confident they are not in use by any installation.
Removing the CRDs will cause your existing Ambassador Mappings and other resources to be deleted as well.
Run 

`kubectl delete crd -l product=aes`

to remove any existing AES CRD's.

To reinstall AES, see:

https://www.getambassador.io/docs/latest/tutorials/getting-started/
