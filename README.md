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


### Parameters

`k8ify` has 2 parameters: `environment` and `ref`. Both are optional.


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


### Labels

`k8ify` supports configuring services by using compose labels.

| Label  | Effect  |
| ------ | ------- |
| `k8ify.singleton: true`  | Compose service is only deployed once per environment instead of once per `$ref` per environment  |
| `k8ify.expose: $host`  | The first port is exposed to the internet via a HTTPS ingress with the host name set to `$host`  |
| `k8ify.expose.$port: $host`  | The port `$port` is exposed to the internet via a HTTPS ingress with the host name set to `$host`  |


## Conversion

_Details about the conversion process will be documented here once we have a first somewhat stable implementation._


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
