# Storage

## Basics

K8s volumes can be "RWO" ("Read Write Once") or "RWX" ("Read Write Many"), although the latter isn't supported on all K8s instances. "RWO" volumes can only be used by one replica of an application (Pod) at a time.

K8s supports "Deployments" which can consist of multiple replicas of the same application (Pods), all sharing the same volume, which means that "Deployments" can only be used with "RWX" volumes. The alternative is a "StatefulSet", which works differently: In a StatefulSet each replica of the application (Pod) gets its own dedicated volume, i.e. if you have two replicas you also have two volumes. Therefore StatefulSets can be used with "RWO" volumes because volume sharing doesn't occur.

Volumes can also be shared between entirely different applications. Also, Compose volumes and Compose services can be labeled as singletons. k8ify needs to handle all those cases correctly.


## PersistentVolumeClaims vs PersistentVolumeClaim templates

A PersistentVolumeClaim ("PVC") is a request to K8s to provide a Volume, and the existence (or absence) of a PersistentVolumeClaim triggers code to create or delete Volumes. Deployments and StatefulSets work very differently when it comes to PersistentVolumeClaims, resp. who creates them.

With a Deployment the Volumes needed don't change if you scale the Deployment. A Deployment with 10 replicas (Pods) has exactly the same Volume(s) as a Deployment with 1 replica because the Volumes are shared between all replicas. Therefore the PersistentVolumeClaims are static and have to be created by the K8s user (or in this case k8ify), just like any other resource.

With a StatefulSet the Volumes change when it scales; a StatefulSet with 10 replicas needs all the Volumes for each replica separately. Therefore the PersistentVolumeClaims used by a StatefulSet can't be static, they must be created dynamically as the StatefulSet scales. Therefore the PersistentVolumeClaims for StatefulSets are defined as a template inside the StatefulSet, just like a container template. (Note that a StatefulSet can use both RWO Volumes whose PVCs are created via template and RWX Volumes whose PVCs were created directly at the same time)

k8ify handles all of these cases.


## Principles of Operation

* Ignore any bind mounts (used for development and not relevant in K8s)
* Volumes are RWO by default (not shared)
* If a Compose service uses one or more non-shared Volume(s) (RWO), the service will be translated to a StatefulSet
* If a Compose service uses no Volumes or all of them are marked as shared (RWX), the service will be translated to a Deployment
* Impossible combinations (e.g. RWO Volume used by multiple Compose services) are detected and reported to the user


## Error Cases

- Compose service uses a volume that doesn't exist
- Multiple Compose services use same volume but the volume is not configured as `shared`
- `k8ify.singleton` label differs between a Compose service and its volumes


## Volume Labels

### `k8ify.shared`

If `true` create a PVC and set its AccessMode to `ReadWriteMany` (RWX).

### `k8ify.singleton`

If `true` and a PVC is created then omit the `refSlug` from its name.

### `k8ify.storageClass`

This sets the `spec.storageClassName` field of the PVC. This is useful in some cases to choose between ssd and hdd storage or to enable encryption.
