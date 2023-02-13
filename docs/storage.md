# Storage

Principles:

* Storage is never shared by default
* Ignore any bind-mounts


## Concepts

Relevant for both volumes and services is the `k8ify.singleton` label.

Relevant for the volume is the `k8ify.shared` label.


## Volumes

By default, create no PVC and set the AccessMode to `ReadWriteOnce`.

### `k8ify.shared`

If `true`, create a PVC and set its AccessMode to `ReadWriteMany`.


### `k8ify.singleton`

If `true`, and a PVC is created, omit the `refSlug` from its name.


## Services

By default, create a Deployment.

If any `ReadWriteOnce` volumes are attached, create a StatefulSet instead and include all `ReadWriteOnce` volumes in the volume templates.


### `k8ify.singleton`

If `true`, omit the `refSlug` from its name.


## Error cases

- Service uses a volume that doesn't exist
- Service is singleton, but volume isn't.
- Multiple services use same volume, but it's `ReadWriteOnce`
- `k8ify.singleton` label differs between service and its volumes.