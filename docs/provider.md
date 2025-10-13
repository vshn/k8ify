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

## Plain LoadBalancer Scheme

To expose k8s Services of type LoadBalancer directly for non HTTP traffic (without an Openshift route or an ingress), some providers need additional manifests
or configuration.
This can be used to expose a database.

### appuio-cloudscale

On APPUiO Cloudscale, you need a `CiliumNetworkPolicy` to expose a plain k8s Service of type LoadBalancer. Refer to [Change to LoadBalancer on APPUiO](https://github.com/appuio/appuio-cloud-community/discussions/60) for
details.

To enable this, set `x-targetCfg.exposePlainLoadBalancerScheme` to
`appuio-cloudscale`:

```yaml
x-targetCfg:
  exposePlainLoadBalancerScheme: appuio-cloudscale
```

For a full example, see [compose.yml: expose plain on appuio](/tests/golden/expose-plain-appuio/compose.yml).
