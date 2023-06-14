# Provider Specific Functionality

While k8ify tries to be as provider-agnostic as possible, some functionality depends on how k8s is set up and which operators are available.

Examples:

* TLS support for Ingresses depends on the availability of [cert-manager](https://cert-manager.io/)
* Support for encrypted volumes ("bring your own encryption") may be available, but works completely differently depending on the k8s setup

To have support for such features a target provider can be specified using the `--provider` flag. This page documents the available providers and their features.

## appuio-cloudscale

This provider supports extra features of the [APPUiO](https://www.appuio.ch/) sites running on Cloudscale.

Enable this functionality by specifying the k8ify parameter `--provider appuio-cloudscale`.

### [LUKS Encrypted Volumes](https://hub.syn.tools/csi-cloudscale/index.html)

Example:
```
volumes:
  mongodb-data:
    labels:
      k8ify.storageClass: ssd-encrypted # or bulk-encrypted
```

The storage classes "ssd-encrypted" or "bulk-encrypted" will trigger the LUKS encryption support.

You need to specify the LUKS encryption keys via environment variables. The name of the variables depend on the names of the volumes and environments; k8ify will print an error message with the name of a variable if it does not exist. Be sure to restrict the visibility of those environment variables appropriately. The name of these environment variables depends on the name of the volumes and potentially the name of the services; k8ify will tell you how to set them up. LUKS encryption keys should have an entropy of at least 512 bits, but they can be longer. Suitable keys can be generated e.g. via `pwgen -s 100 1`.
