# k8ify

`k8ify` converts [**Compose**][Compose] files into **Kubernetes** manifests.

This project adheres to [Semantic Versioning][SemVer] and tries to not break functionality between major versions.

## Breaking Changes

First things first.

### v1 to v2
* We've upgraded the compose-go library from v1 to v2. This can affect parsing of the compose file; in particular v1 sorted arrays while parsing and v2 keeps the ordering as it is, which can affect "ports" and "volumes" arrays. Please check the resulting manifests for changes before applying them to your cluster. If necessary change the order of array elements in your compose files to match the previous output.


## Goal & Purpose

The purpose of this project is to allow developers to generate state-of-the-art Kubernetes manifests without the need to have a degree in Kubernetes-Manifestology.

Just by describing the components of their application they should be able to get a set of Kubernetes manifests that adhere to best practices.

The spiritual prototype for k8ify is [Kompose][] and k8ify tries to do things similarly to Kompose. Unfortunately Kompose does not provide the flexibility we need, hence the custom implementation.

We chose the [Compose][] format as many developers already use Docker Compose for local development and are familiar with it.


### Non-Goals

Out of scope of this project are:


#### Builds

Building their applications (and container images thereof) is something most developers are very proficient in and don't really need any help with. Furthermore the build and test processes are usually very custom to the application.


#### Deployments

The idea is that the "build" stage of a deployment pipeline generates the manifests and outputs a diff by comparing the manifests to the state in the target cluster (e.g. using `kubectl diff`), and the "deploy" stage then applies the manifests.
This results in flexibility to support various modes of deployment, be it plain `kubectl apply` in the next step of the CI/CD pipeline or a GitOps solution like ArgoCD or FluxCD.


## Mode of Operation

`k8ify` takes Compose files in the current working directory and converts them to Kubernetes manfests. The manifests are written to the `manifests` directory.


### Command Line Arguments

`k8ify` supports the following command line arguments:

- Argument #1: `environment`. Optional.
- Argument #2: `ref`. Optional.
- `--modified-image [IMAGE]`: IMAGE has changed. Optional, repeatable.
- `--shell-env-file [FILENAME]`: Load additional shell environment variables from file. Optional, repeatable.

#### `environment`

`k8ify` supports different environments (e.g. "prod", "test" and "dev") by merging Compose files. A setup could look like this:

- `compose.yml` - The global default, the base used by all environments.
- `compose-prod.yml` - Additional information about the `prod` environment. Used by `k8ify` when asked to generate manifests for the `prod` environment.
- `compose-test.yml` - Additional information about the `test` environment. Used by `k8ify` when asked to generate manifests for the `test` environment.
- `compose-dev.yml` - Additional information about the developer's local `dev` environment. Never used by `k8ify` but used by developer for running everything locally.

`k8ify` will choose the correct Compose files and merge them based on the selected environment.


#### `ref` - Multiple deployments in the same environment

`k8ify` supports multiple deployments in the same environment, e.g. to deploy different branches of an application into the same `test` environment. It does so by adding a `-$ref` suffix to the name of all generated resources.

Each Compose service is, by default, deployed for each `ref`. If you want to deploy a service only once per environment (e.g. a single shared database for all deployments) you can do so by adding the `k8ify.singleton: true` label to the service.

A resulting deployment might look like this:

- deployment/service `mongodb` (singleton) with secret `mongodb-env`
- deployment/service `myapp-testbranch1` with secret `myapp-testbranch1-env`
- deployment/service `myapp-testbranch2` with secret `myapp-testbranch2-env`

#### `--modified-image [IMAGE]` - Handling image rebuilding with the same image tag

A build pipeline usually builds at least one image, tags it with a version number or branch name and pushes it to a registry. If the tag is new, the Deployments/ReplicaSets using this image get updated with the new version number. This will cause K8s to restart the corresponding Pods in order for them to use the new image.

However, if the image tag stays the same (which is often the case for test branches) there is a problem: The Deployment/ReplicaSet does not need to change at all, and if there is no change K8s does not roll out the new image.

To work around this problem k8ify can introduce a dummy change to the Deployment/ReplicaSet to force the roll-out. In order to identify which Deployments/ReplicaSets need this dummy change, you can tell k8ify which images have been rebuilt and k8ify will automatically find the relevant Deployments/ReplicaSets.

This parameter is generally set by the CI/CD pipeline, because the pipeline knows which images it has generated in earlier steps. The image should be specified as `$SERVICE:$TAG` or `$NAMESPACE/$SERVICE:$TAG`, depending on how specific you need to be. You can repeat this parameter for any number of images.

#### `--shell-env-file [FILENAME]` - Load additional shell environment variables from file

k8ify relies on the shell environment to fill placeholders in the Compose files. This argument can be used to load additional variables. The files have the usual "KEY=VALUE" format and they support quoted values.

A use case could be to protect your secrets. Instead of loading them into the shell environment you could put them into a file and use this argument to load said file.


### Labels

`k8ify` supports configuring services and volumes by using Compose labels. All labels are optional.

#### General

Service Labels

| Label  | Effect  |
| ------ | ------- |
| `k8ify.imagePullSecret: $env_var` | Creates an ImagePullSecret from the referenced environment variable([example](https://github.com/vshn/k8ify/blob/22b814f1ee0e7c0be2b8788096702678f359a71b/tests/golden/imagepullsecrets.yml)). Can be used once per service.|
| `k8ify.singleton: true`  | Compose service is only deployed once per environment instead of once per `$ref` per environment  |
| `k8ify.expose: $host`  | The first port is exposed to the internet via a HTTPS ingress with the host name set to `$host`  |
| `k8ify.expose.$port: $host`  | The port `$port` is exposed to the internet via a HTTPS ingress with the host name set to `$host`  |
| `k8ify.converter: $script`  | Call `$script` to convert this service into a K8s object, expecting YAML on `$script`'s stdout. Used for plugging additional functionality into k8ify. The first argument sent to `$script` is the name of the resource, after that all the parameters follow (next row) |
| `k8ify.converter.$key: $value`  | Call `$script` with parameter `--$key $value` |
| `k8ify.serviceAccountName: $name`  | Set this service's pod(s) spec.serviceAccountName to `$name`, which tells the pod(s) to use ServiceAccount `$name` for accessing the K8s API. This does not set up the ServiceAcccount itself. |
| `k8ify.partOf: $name`  | This Compose service will be combined with another Compose service (resulting in a deployment or statefulSet with multiple containers). Useful e.g. for sidecars or closely coupled services like nginx & php-fpm. |
| `k8ify.annotations.$key: $value`  | Add annotation(s) to all resources generated by k8ify |
| `k8ify.$kind.annotations.$key: $value`  | Add annotation(s) to specific resource types generated by k8ify. $kind uses the default case used by k8s and is always singular (e.g. "StatefulSet") |
| `k8ify.exposePlain.$port: true`  | Set up a k8s Service which exposes this port directly instead of using the cluster-wide reverse proxy/load balancer, useful for non-HTTP applications (for HTTP always use `k8ify.expose`). Allocated public IP is visible in the k8s Service's `.status` field. |
| `k8ify.exposePlain.$port.type: ClusterIP\|LoadBalancer\|ExternalName\|NodePort`  | Set the k8s Service type (default `LoadBalancer`) |
| `k8ify.exposePlain.$port.externalTrafficPolicy: Cluster\|Local`  | Set the k8s Service traffic policy (default `Local`). `Local` makes the client IP visible to the application but may provide worse load balancing than `Cluster`. |
| `k8ify.exposePlain.$port.healthCheckNodePort: $port`  | Set the k8s Service health check port number. |
| `k8ify.enableServiceLinks: $value` | Inject ENV variables for each K8s service in the namespace. |

Volume Labels

| Label  | Effect  |
| ------ | ------- |
| `k8ify.size: 10G`  | Requested volume size. Defaults to `1G`.  |
| `k8ify.singleton: true`  | Volume is only created once per environment instead of once per `$ref` per environment  |
| `k8ify.shared: true` | Instead of `ReadWriteOnce`, create a `ReadWriteMany` volume; Services with multiple replicas will all share the same volume  |
| `k8ify.storageClass: ssd` | Specify the storage class, e.g. 'hdd' or 'ssd'. Available values depend on the target system. |

#### Health Checks

For each Compose service k8ify will set up a basic TCP based health check (liveness and startup) by default.
For all services providing HTTP endpoints you should provide at least a basic health check path and point `k8ify.liveness` to it.
This replaces the TCP based health check by a more specific HTTP(S) check.

| Label  | Effect  |
| ------ | ------- |
| `k8ify.liveness` | Configure a container liveness check. If the check fails the container will be restarted. |
| `k8ify.liveness: $path` | Configure the path for a HTTP GET based liveness check. Default is "", which disables the HTTP GET check and uses a simple TCP connection check instead. |
| `k8ify.liveness.path: $path` | See previous |
| `k8ify.liveness.enabled: true` | Enable or disable the liveness check. Default is true. |
| `k8ify.liveness.scheme: 'HTTP'` | Switch to HTTPS for HTTP GET based liveness check. Default is HTTP. |
| `k8ify.liveness.periodSeconds: 30` | Configure the periodicity of the check. Default is 30. |
| `k8ify.liveness.timeoutSeconds: 60` | Configure the timeout of the check. Default is 60. |
| `k8ify.liveness.initialDelaySeconds: 0` | Delay before the first check is executed. Default is 0. |
| `k8ify.liveness.successThreshold: 1` | Number of times the check needs to succeed in order to signal that everything is fine. Default is 1. |
| `k8ify.liveness.failureThreshold: 3` | Number of times the check needs to fail in order to signal a failure. Default is 3. |
| `k8ify.startup` | Configure a container startup check. This puts the liveness check on hold during container startup in order to prevent the liveness check from killing it. |
| `k8ify.startup.*` | All the settings work the same as for `k8ify.liveness`. **The values are copied over from `k8ify.liveness` by default** with the following exceptions: |
| `k8ify.startup.periodSeconds: 10` | In order to have quick startup the default is lowered to **10**. |
| `k8ify.startup.failureThreshold: 30` | In order to give the application a total of 300 seconds to start up, the default is raised to **30**. |
| `k8ify.readiness` | This check decides if traffic should be sent to this instance or not. In contrast to the liveness check a failing readiness check will not restart the pod, just mark it as unavailable and not send traffic to it. |
| `k8ify.readiness.*` | All the sub-values work the same as for `k8ify.liveness` incl. defaults. No values are copied over. However the readiness check is disabled by default. |
| `k8ify.readiness.enabled: false` | Enable or disable the readiness check. Default is false. |

#### Prometheus ServiceMonitor

If the `k8ify.prometheus.serviceMonitor` label is set to true for a service, a [Prometheus ServiceMonitor](https://prometheus-operator.dev/docs/api-reference/api/#monitoring.coreos.com/v1.ServiceMonitor) manifest will be emitted.
By default, the first port is used as endpoint.

| Label  | Effect                                                                                           |
|--------|--------------------------------------------------------------------------------------------------|
| `k8ify.prometheus.serviceMonitor: true` | Emit a ServiceMonitor manifest if `true`. Default is `false`. |
| `k8ify.prometheus.serviceMonitor.interval: 30s` | Interval to use. Default is `30s`. |
| `k8ify.prometheus.serviceMonitor.path: /actuator/metrics` | Path to use. Default is `/actuator/metrics`. |
| `k8ify.prometheus.serviceMonitor.scheme: http` | Scheme to use. Default is `http`. |
| `k8ify.prometheus.serviceMonitor.endpoint.name: 8080` | Port to use for ServiceMonitor. References the published port number. Default is the first port. |
| [BasicAuth](https://prometheus-operator.dev/docs/api-reference/api/#monitoring.coreos.com/v1.BasicAuth) | |
| `k8ify.prometheus.serviceMonitor.endpoint.basicAuth: true` | Enable BasicAuth for Endpoint. Default is `false` |
| `k8ify.prometheus.serviceMonitor.endpoint.basicAuth.username: "username"` | Username for BasicAuth for Endpoint. |
| `k8ify.prometheus.serviceMonitor.endpoint.basicAuth.password: "password"` | Password for BasicAuth for Endpoint. |
| [TlsConfig](https://prometheus-operator.dev/docs/api-reference/api/#monitoring.coreos.com/v1.SafeTLSConfig) | |
| `k8ify.prometheus.serviceMonitor.endpoint.tlsConfig: "true"` | Enable TLS configuration for the endpoint. Default is `false`. |
| `k8ify.prometheus.serviceMonitor.endpoint.tlsConfig.ca: ${SERVICE_MONITOR_CA}` | CA certificate for TLS. |
| `k8ify.prometheus.serviceMonitor.endpoint.tlsConfig.cert: ${SERVICE_MONITOR_CERT}` | Client certificate for TLS. |
| `k8ify.prometheus.serviceMonitor.endpoint.tlsConfig.insecureSkipVerify: "false"` | Whether to skip TLS certificate verification. |
| `k8ify.prometheus.serviceMonitor.endpoint.tlsConfig.keySecretValue: ${SERVICE_MONITOR_KEY}` | Client key for TLS, typically a reference to a secret. |
| `k8ify.prometheus.serviceMonitor.endpoint.tlsConfig.maxVersion: "TLS13"` | Maximum TLS version to use. Options are `TLS10`, `TLS11`, `TLS12`, `TLS13`. |
| `k8ify.prometheus.serviceMonitor.endpoint.tlsConfig.minVersion: "TLS13"` | Minimum TLS version to use. Options are `TLS10`, `TLS11`, `TLS12`, `TLS13`. |

#### Target Cluster Configuration

There are some cases in which the output of k8ify needs to be different based on the target cluster's configuration. To make this work some properties of the target cluster can be configured via the `x-targetCfg` root key in the Compose file.

| Key  | Effect  |
| ---- | ------- |
| `appsDomain: $domain`  | A cluster may have a wildcard certificate for apps to use. If you configure this option and expose a service using `$domain`, the resulting Ingress uses this wildcard certificate (instead of e.g. Let's Encrypt). |
| `maxExposeLength: $length`  | k8ify does a length check on the exposed domain names, because if they're too long the Ingress will not work. Default is 63.  |
| `encryptedVolumeScheme: $provider`  | The implementation of encrypted volumes is provider specific. Use this to enable support for a provider. See [Provider](./docs/provider.md) for more information.  |
| `exposePlainLoadBalancerScheme: $provider`  | Certain provider need extra manifests to expose a plain k8s Service of type LoadBalancer. See [Provider](./docs/provider.md) for more information.  |


## Conversion

The conversion process is documented in depth in [Conversion](./docs/conversion.md).


## Storage

Storage support is documented in depth in [Storage](./docs/storage.md).


## Testing

In order to validate that `k8ify` does what we expect it to do, we use the concept of "golden tests": a predefined set of inputs (Compose files) and outputs (Kubernetes manifests) are added to the repository. During the testing process we run `k8ify` against each of the inputs, and verify that the outputs match the expected outputs.

To set up a golden test named `$NAME`, you need to create two things in the `tests/golden/` directory:

1. A file called `$NAME.yml`, and
2. A directory called `$NAME` containing Compose files.

The structure of the YAML file should look like this:

```yaml
environments:
  prod: {}
  test:
    refs:
      - foo
      - bar
    vars:
      SOME_ENV_VAR: "true"
      ANOTHER_ENV_VAR: "42"
```

Note that both the `refs` and `vars` fields are optional, but allow you to control the tests:

- `refs` will make the test run `k8ify` for each of the provided values. If no `refs` is defined, `k8ify` will be run once for this environment with an empty `ref` value.
- `vars` can contain environment variables that are ADDED to the ones that are already set within the testing environment. If your compoose file makes use of any environment variables, make sure to add them here for reproducibility.

To actually run the tests run `go test` in the root of the repository.


## License

This project is licensed under the [BSD 3-Clause License](LICENSE)

[Compose]: https://github.com/compose-spec/compose-spec/blob/master/spec.md
[Kompose]: https://kompose.io/
[SemVer]: https://semver.org/
