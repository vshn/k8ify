# Provider Specific Functionality

While k8ify tries to be as provider agnostic as possible, some functionality depends on how k8s is set up and which operators are available.


## Encrypted Volume Scheme

How encrypted volumes have to be set up depends on the target k8s setup. Therefore k8ify allows you to specify which encryption scheme to use via the `x-targetCfg` option.

Example:
```
x-targetCfg:
  encryptedVolumeScheme: appuio-cloudscale
```

### appuio-cloudscale

Usage of encrypted volumes on APPUiO Cloudscale is documented at [LUKS Encrypted Volumes](https://hub.syn.tools/csi-cloudscale/index.html), but with k8ify you should not need to worry about this.

Example:
```
volumes:
  mongodb-data:
    labels:
      k8ify.storageClass: ssd-encrypted # or bulk-encrypted
```

The storage classes "ssd-encrypted" or "bulk-encrypted" trigger the LUKS encryption support.

k8ify expects you to specify the LUKS encryption keys via environment variables. These environment variables must have specific names, which depend on the names of the volumes and environments. The easiest way to set this up is to run k8ify without these variables set, and it will print error messages containing the names of variables it expects.

LUKS encryption keys should have an entropy of at least 512 bits. Suitable keys can be generated e.g. via `pwgen -s 100 1`. Be sure to restrict the visibility of environment variables containing LUKS keys appropriately.
