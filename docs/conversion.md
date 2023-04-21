# Conversion

This document describes the conversion process `k8ify` applies to convert Compose files to Kubernetes manifests.

Compose files define "compose services". Every compose service is translated into Kubernetes resources individually.

In general, every compose service is implemented as a Deployment (or StatefulSet, see below), exposed ports are exposed as Services, volumes are mapped to Persistent Volume Claims (which make Kubernetes provide Persistent Volumes) and environment variables are saved into secrets. Services may further be exposed via Ingresses.

This results in the following list of Kubernetes resource for each compose service:

* 1 Workload ([`Deployment`](#k8s-deployment) or [`StatefulSet`](#k8s-statefulset))
* 0-1 [`Service`](#k8s-service) (a single service can cover multiple ports; if no ports are exposed no service is created)
* 1 [`Secret`](#k8s-secret) (may be empty)
* 0-n [`PersistentVolumeClaim`](#k8s-persistentvolumeclaim) (optionally one per volume)
* 0-n [`Ingress`](#k8s-ingress) (one per port, IF enabled via `k8ify.expose` label on the compose service)


### Special considerations

Some compose concepts don't match neatly onto Kubernetes concepts, so some special care must be taken.

#### Ingress

In order to make a service available to the outside world, we need to support Ingresses. However, compose files have no notion of "available to the outside world", hence there is no direct way of generating an Ingress from the data in a compose file. Hence, setting up Ingresses is implemented via compose service labels (see [Labels](../README.md#labels)).


#### Environment variables and Secrets

Compose supports secrets and environment variables. However, the "secret" support is limited to file-based secrets, which are inherently incompatible with the Twelve-Factor Application principles, thus we don't want to use these.

Instead, we only use the "environment" functionality of compose. However, environment variables can contain sensitive information, and we don't know which ones do and which ones don't. Thus, the conversion does not store any environment variables in the deployment, but puts a secretRef in there and then writes all environment variables to one secret per compose service.


#### Volumes

Only volumes defined in the `volumes` topl level section of the Compose files are taken into consideration. Local bind mounts or `tmpfs` mounts are ignored.

By default Volumes will be assigned the `ReadWriteOnce` access mode to prevent multiple instances of an application writing to the same storage location.

This chan be changed to `ReadWriteMany` by adding the label `k8ify.shared: true` to the volume.


#### Deployments vs. StatefulSets

By default, a compose service will be translated into a `Deployment`.

If the compose service has non-shared (`ReadWriteOnce`) volumes mounted, a `StatefulSet` is used instead. This results in every replica getting its own `ReadWriteOnce` PersistentVolume.

If a compose service only uses shared (`ReadWriteMany`) volumes, it will still be translated into a `Deployment`.


#### Labels

Note that K8ify works with both Labels on Compose services (set in Compose files under `services.$name.labels`) and Kubernetes resource labels (set on individual resources under `metadata.labels`).
Labels are not automatically copied from Compose files to Kubernetes manifests.
Instead, the two concepts are used for different purposes:

Labels on **Compose services** and **volumes** are used to **configure** and customize manifest generation.

Labels on the generated **Kubernetes resources** are used to **identify** resources managed via K8ify.

The following set of labels is applied to all generated Kubernetes resources:

```yaml
labels:
  # `$name`
  k8ify.service: "myapp"
  # `$refSlug`
  k8ify.ref-slug: "feat-foo"
```


## Conversion Table

The following variables will be used in the table below:

* `$name` - Name of the Compose service (eg the "key" in the `services` map in the Compose file), e.g. **myapp**
* `$ref` - Name of the `ref` passed to `k8ify`, see [Parameters](../README.md#parameters), e.g. **feat/foo**
* `$refSlug` - Normalized version of `ref` that is a valid DNS label (and hence can be used as a Kubernetes label value), eg **feat-foo**

See [Example input](#example-input) below on how a Compose file would have to look to generat the following example manifests.

Note that some fields were omited here for brevity.


### Common

Labels are documented [above](#labels) and not repeated in the examples below.


### K8s Deployment

See [Deployments vs. Statefulsets](#deployments-vs-statefulsets) for a documentation when a Deployment is used, and when a [StatefulSet](#k8s-statefulset).

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  # If singleton or no `ref` given: "$name"
  # Otherwise: "$name-$refSlug"
  name: "myapp-feat-foo"  # or "myapp"
spec:
  # `services.$name.deploy.replicas`, defaults to `nil`
  replicas: 2
  strategy:
    # Depending on `services.$name.deploy.update_config.order`:
    # * `stop-first` (default) -> `Recreate`
    # * `start-first` -> `Rolling`
    type: Recreate
  template:
    metadata:
      annotations:
        # If this Deployment uses one of the images that have been flagged as
        # modified via the --modified-image argument, this is set to the current
        # timestamp to ensure restarts of all pods
        k8ify.restart-trigger: "1675680748"
    spec:
      containers:
          # If singleton or no ref given: `$name`, otherwise: `$name-$refSlug`
        - name: "myapp-feat-foo"  # or "myapp"
          # `services.$name.image`
          # Note: We support compose files per environment, so the image can be
          # configured there. These compose files also support substitution of
          # env vars that are set by the CI system, e.g. to fill in the correct
          # tag name.
          image: "docker.io/mycorp/myapp:v0.5.7"
          envFrom:
            - secretRef:
                # `$name(-$refSlug)-env`
                name: "myapp-feat-foo-env"
          # List of the target port values from `services.$name.ports`
          ports:
            - containerPort: 8000
            - containerPort: 9000
          # List of all volume mounts from `service.$name.volumes`:
          # * `mountPath` is the part behind ":"
          # * `name` is the sanitized volume name: `myapp-data`
          volumeMounts:
            - mountPath: /data
              name: myapp-data
          # By default both a livenessProbe and startupProbe are set up.
          # `services.$name.labels["k8ify.liveness"]` and sub-labels
          livenessProbe:
            failureThreshold: 3
            # `httpGet` if `services.$name.labels["k8ify.liveness"]` or `services.$name.labels["k8ify.liveness.path"]` is set, `tcpSocket` otherwise
            httpGet:
              # `services.$name.labels["k8ify.liveness"]` or `services.$name.labels["k8ify.liveness.path"]`
              path: /health
              # `services.$name.labels["k8ify.liveness.port"]` if it exists, otherwise first containerPort
              port: 8000
              # `services.$name.labels["k8ify.liveness.scheme"]`
              scheme: HTTP
            # `services.$name.labels["k8ify.liveness.periodSeconds"]`
            periodSeconds: 30
            # `services.$name.labels["k8ify.liveness.successThreshold"]`
            successThreshold: 1
            # `services.$name.labels["k8ify.liveness.timeoutSeconds"]`
            timeoutSeconds: 60
          # `services.$name.labels["k8ify.startup"]` and sub-labels
          startupProbe:
            failureThreshold: 30
            httpGet:
              path: /health
              port: 8000
              scheme: HTTP
            periodSeconds: 10
            successThreshold: 1
            timeoutSeconds: 60
          resources:
            limits:
              # `services.$name.deploy.resources.limits.cpu`, defaults to reservations * 10
              cpu: "1"
              # `services.$name.deploy.resources.limits.memory`, defaults to same value as reservations
              memory: 256Mi
            requests:
              # `services.$name.deploy.resources.reservations.cpu`
              cpu: 100m
              # `services.$name.deploy.resources.reservations.memory`
              memory: 256Mi
          # `services.$name.entrypoint`, overwrites 'ENTRYPOINT' in Dockerfile
          command: ["echo"]
          # `services.$name.command`, overwrites 'CMD' in Dockerfile
          args: ["Hello World"]
          # hard-coded
          imagePullPolicy: Always
          # `services.$name.labels["k8ify.serviceAccountName"], not set by default
          serviceAccountName: "myappk8saccess"
      # Values from `services.$name.volumes`, translated as the volumeMounts above
      volumes:
        - name: myapp-data
          persistentVolumeClaim:
            claimName: myapp-data
```


### K8s StatefulSet

See [Deployments vs. Statefulsets](#deployments-vs-statefulsets) for a documentation when a [Deployment](#k8s-deployment) is used, and when a StatefulSet.

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  # If singleton or no `ref` given: "$name"
  # Otherwise: "$name-$refSlug"
  name: "myapp-feat-foo"  # or "myapp"
spec:
  # `services.$name.deploy.replicas`, defaults to `nil`
  replicas: 2
  template:
    metadata:
      annotations:
        # If this StatefulSet uses one of the images that have been flagged as
        # modified via the --modified-image argument, this is set to the current
        # timestamp to ensure restarts of all pods
        k8ify.restart-trigger: "1675680748"
    spec:
      containers:
          # If singleton or no ref given: `$name`, otherwise: `$name-$refSlug`
        - name: "myapp-feat-foo"  # or "myapp"
          # `services.$name.image`
          # Note: We support compose files per environment, so the image can be
          # configured there. These compose files also support substitution of
          # env vars that are set by the CI system, e.g. to fill in the correct
          # tag name.
          image: "docker.io/mycorp/myapp:v0.5.7"
          envFrom:
            - secretRef:
                # `$name(-$refSlug)-env`
                name: "myapp-feat-foo-env"
          # List of the target port values from `services.$name.ports`
          ports:
            - containerPort: 8000
            - containerPort: 9000
          # List of all volume mounts from `service.$name.volumes`:
          # * `mountPath` is the part behind ":"
          # * `name` is the sanitized volume name: `myapp-data`
          volumeMounts:
            - mountPath: /data
              name: myapp-data
          # By default both a livenessProbe and startupProbe are set up.
          # `services.$name.labels["k8ify.liveness"]` and sub-labels
          livenessProbe:
            failureThreshold: 3
            # `httpGet` if `services.$name.labels["k8ify.liveness"]` or `services.$name.labels["k8ify.liveness.path"]` is set, `tcpSocket` otherwise
            httpGet:
              # `services.$name.labels["k8ify.liveness"]` or `services.$name.labels["k8ify.liveness.path"]`
              path: /health
              # `services.$name.labels["k8ify.liveness.port"]` if it exists, otherwise first containerPort
              port: 8000
              # `services.$name.labels["k8ify.liveness.scheme"]`
              scheme: HTTP
            # `services.$name.labels["k8ify.liveness.periodSeconds"]`
            periodSeconds: 30
            # `services.$name.labels["k8ify.liveness.successThreshold"]`
            successThreshold: 1
            # `services.$name.labels["k8ify.liveness.timeoutSeconds"]`
            timeoutSeconds: 60
          # `services.$name.labels["k8ify.startup"]` and sub-labels
          startupProbe:
            failureThreshold: 30
            httpGet:
              path: /health
              port: 8000
              scheme: HTTP
            periodSeconds: 10
            successThreshold: 1
            timeoutSeconds: 60
          resources:
            limits:
              # `services.$name.deploy.resources.limits.cpu`, defaults to reservations * 10
              cpu: "1"
              # `services.$name.deploy.resources.limits.memory`, defaults to same value as reservations
              memory: 256Mi
            requests:
              # `services.$name.deploy.resources.reservations.cpu`
              cpu: 100m
              # `services.$name.deploy.resources.reservations.memory`
              memory: 256Mi
          # `services.$name.entrypoint`, overwrites 'ENTRYPOINT' in Dockerfile
          command: ["echo"]
          # `services.$name.command`, overwrites 'CMD' in Dockerfile
          args: ["Hello World"]
          # hard-coded
          imagePullPolicy: Always
          # `services.$name.labels["k8ify.serviceAccountName"], not set by default
          serviceAccountName: "myappk8saccess"
      # See PersistentVolumeClaim below for how the values are generated.
      volumeTemplates:
        - metadata:
            name: myapp-data
          spec:
            accessModes:
              - ReadWriteOnce
            resources:
              requests:
                # Value of the `k8ify.size` label on the volume
                storage: 10Gi
```


### K8s Service

```yaml
apiVersion: v1
kind: Service
metadata:
  # If singleton or no `ref` given: "$name", otherwise: "$name-$refSlug"
  name: "myapp-feat-foo"  # or "myapp"
spec:
  # `services.$name.ports`, but translated like this:
  # * `name` is the first port number as a string
  # * `port` is the first port number as an int
  # * `targetPort` is the second port number as an int
  ports:
  - name: "8001"
    port: 8001
    targetPort: 8000
  - name: "9001"
    port: 9001
    targetPort: 9000
  # Same as `metadata.labels`
  selector:
    k8ify.service: "myapp"
    k8ify.ref-slug: "feat-foo"
```


### K8s Secret

```yaml
apiVersion: v1
kind: Secret
metadata:
  # If singleton or no `ref` given: "$name-env"
  # Otherwise: "$name-$refSlug-env"
  name: "myapp-feat-foo-env"  # or "myapp-env"
# All values from `services.$name.environment`, split by "=" and put into a
# map
stringData:
  mongodb_database: myapp
  mongodb_hostname: mongo
```


### K8s PersistentVolumeClaim

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  # See `Deployment` or `StatefulSet` for how volumes and volume claims are
  # named
  name: "myapp-feat-foo"  # or "myapp"
spec:
  # If "k8ify.shared" (via label): "ReadWriteMany"
  # Otherwise: "ReadWriteOnce"
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      # Value of the `k8ify.size` label on the volume
      storage: 10Gi
```


### K8s Ingress

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  # `$name(-$ref)-$portString`
  # where `$portString` is the same as `spec.ports.$i.name` in the referenced
  # K8s service
  name: myapp-feat-foo-8001
  # Whatever is configured in the config file (`.k8ify.default.yaml`) under
  # `ingressPatch.addAnnotations`
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-production
spec:
  rules:
      # `services.$name.labels["k8ify.expose"]`, or
      # `services.$name.labels["k8ify.expose.$port"]`, or
    - host: myapp.example.com
      http:
        paths:
          - backend:
              service:
                # Whatever the Service is named
                name: "myapp-feat-foo"
                port:
                  # Value from corresponding service & port
                  number: 8001
            # hard-coded
            path: /
            # hard-coded
            pathType: Prefix
  tls:
      # Same as `host` above
    - hosts:
        - portal-k8ify.apps.cloudscale-lpg-2.appuio.cloud
      # Ingress name suffixed by "-tls"
      secretName: portal-mpi-8001-tls
```


## Example Input

The manifests above were generated using the following command:

```sh
k8ify test feat/foo
```

K8ify configuration

```yaml
ingressPatch:
  addAnnotations:
    cert-manager.io/cluster-issuer: letsencrypt-production
```

Compose file

```yaml
services:
  myapp:
    labels:
      k8ify.expose.8001: myapp.example.com
      k8ify.liveness: /health
    image: docker.io/mycorp/myapp:v0.5.7
    deploy:
      replicas: 2
    ports:
      - "8001:8000"
      - "9001:9000"
    volumes:
      - "./:/src"
      - "myapp_data:/data"
    entrypoint: ["echo"]
    command: ["Hello World"]
    environment:
      - mongodb_hostname=mongo
      - mongodb_database=$MONGO_DB

volumes:
  myapp_data:
    labels:
      k8ify.size: 10Gi
```
