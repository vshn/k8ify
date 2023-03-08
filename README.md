# k8ify

`k8ify` converts [**Compose**][Compose] files into **Kubernetes** manifests.

**Warning:** This tool is currently under heavy development. Stuff may change & break between releases!


## Goal & Purpose

The purpose of this project is to allow developers to generate state-of-the-art Kubernetes manifests without the need to have a degree in Kubernetes-Manifestology.

Just by describing the components of their application they should be able to get a set of Kubernetes manifests that adhere to industry best practices.

The spiritual prototype for k8ify is [Kompose][] and k8ify tries to do things similarly to Kompose if possible. Unfortunately Kompose does not provide the flexibility we need, hence the custom implementation.

We choose the [Compose][] format as many developers already use Docker Compose for local development environments and are familiar with the format.


### Non-Goals

Out of scope of this project are:


#### Builds

Building their applications (and container images thereof) is something most developers are very proficient in and don't really need any help with. Furthermore the build and test processes are usually very custom to the application.


#### Deployments

The idea is that the "build" stage of a deployment pipeline generates the manifests and outputs a diff by comparing the manifests to the state in the target cluster (e.g. using `kubectl diff`), and the "deploy" stage then applies the manifests.
This results in flexibility to support various modes of deployment, be it plain `kubectl apply` in the next step of the CI/CD pipeline or a GitOps solution like ArgoCD or FluxCD.


## Mode of Operation

`k8ify` takes compose files in the current working directory and converts them to Kubernetes manfests. The manifests are written to the `manifests` directory.


### Command Line Arguments

`k8ify` supports the following command line arguments:

- Argument #1: `environment`. Optional.
- Argument #2: `ref`. Optional.
- `--modified-image [IMAGE]`: IMAGE has changed. Optional, repeatable.
- `--shell-env-file [FILENAME]`: Load additional shell environment variables from file. Optional, repeatable.

#### `environment`

`k8ify` supports different environments (e.g. "prod", "test" and "dev") by merging compose files. A possible setup could look like this:

- `compose.yml` - The global default, the base used by all environments.
- `compose-prod.yml` - Additional information about the `prod` environment. Used by `k8ify` when asked to generate manifests for the `prod` environment.
- `compose-test.yml` - Additional information about the `test` environment. Used by `k8ify` when asked to generate manifests for the `test` environment.
- `compose-dev.yml` - Additional information about the developers local `dev` environment. Never used by `k8ify` but used by developers for local development.

`k8ify` will choose the correct compose files and merge them based on the selected environment.


#### `ref` - Multiple deployments in the same environment

`k8ify` supports multiple deployments in the same environment, e.g. to deploy different branches of an application into the same `test` environment. It does so by adding a `-$ref` suffix to the name of all generated resources.

Each compose service is, by default, deployed for each `ref`. If you want to deploy a service only once per environment (e.g. a single shared database for all deployments) you can do so by adding the `k8ify.singleton: true` label to the service.

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

k8ify relies on the shell environment to fill placeholders in the compose files. This argument can be used to load additional variables. The files have the usual "KEY=VALUE" format and they support quoted values. 

A use case could be to protect your secrets. Instead of loading them into the shell environment you could put them into a file and use this argument to load said file.


### Labels

`k8ify` supports configuring services and volumes by using compose labels. All labels are optional.

#### General

Service Labels

| Label  | Effect  |
| ------ | ------- |
| `k8ify.singleton: true`  | Compose service is only deployed once per environment instead of once per `$ref` per environment  |
| `k8ify.expose: $host`  | The first port is exposed to the internet via a HTTPS ingress with the host name set to `$host`  |
| `k8ify.expose.$port: $host`  | The port `$port` is exposed to the internet via a HTTPS ingress with the host name set to `$host`  |
| `k8ify.converter: $script`  | Call `$script` to convert this service into a K8s object, expecting YAML on `$script`'s stdout. Used for plugging additional functionality into k8ify. The first argument sent to `$script` is the name of the resource, after that all the parameters follow (next row) |
| `k8ify.converter.$key: $value`  | Call `$script` with parameter `--$key $value` |

Volume Labels

| Label  | Effect  |
| ------ | ------- |
| `k8ify.size: 10G`  | Requested volume size. Defaults to `1G`.  |
| `k8ify.singleton: true`  | Volume is only created once per environment instead of once per `$ref` per environment  |
| `k8ify.shared: true` | Instead of `ReadWriteOnce`, create a `ReadWriteMany` volume; Services with multiple replicas will all share the same volume  |

#### Health Checks

For each compose service k8ify will set up a basic TCP based health check (liveness and startup) by default.
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


## Conversion

The conversion process is documented in-depth in [Conversion](./docs/conversion.md).


## Testing

In order to validate that `k8ify` does what we expect it to do, we use the concept of "golden tests": a predefined set of inputs (compose files) and outputs (Kubernetes manifests) are added to the repository. During the testing process, we run `k8ify` against each of the inputs, and verify that the outputs match the expected outputs.

To set up a golden test named `$NAME`, you need to create two things in the `tests/golden/` directory:

1. A file called `$NAME.yml`, and
2. A directory called `$NAME` containing compose files.

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
- `vars` can contain environment variables that are ADDED to the ones that are already set within the testing environment. If your compoose file makes use of any environment variables, make sure to add them here for reproducability.

To actually run the tests, run `go test` in the root of the repository.


## License

This project is licensed under the [BSD 3-Clause License](LICENSE)

[Compose]: https://github.com/compose-spec/compose-spec/blob/master/spec.md
[Kompose]: https://kompose.io/
