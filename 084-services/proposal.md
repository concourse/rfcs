# Summary

Provide a native way to expose local services to steps.

# Motivation

* Easier integration testing ([concourse/concourse#324](https://github.com/concourse/concourse/issues/324))
  * The current recommended way is to run a privileged `task` with a Docker daemon + `docker-compose` installed, and that task runs `docker-compose up` and the test suite

# Proposal

I propose adding a new `services` field to the `task` step (and eventually `run` step) and special var source `.svc`, e.g.

```yaml
task: integration-tests
file: ci/tasks/test.yml
params:
  POSTGRES_ADDRESS: ((.svc:postgres.address))
  # or
  # POSTGRES_HOST: ((.svc:postgres.host))
  # POSTGRES_PORT: ((.svc:postgres.port))
  # 
  # Services can expose many ports, and each port is named.
  # To access addresses/ports other than the one named 'default', use:
  # ((.svc:postgres.addresses.some-port-name))
  # ((.svc:postgres.ports.some-port-name))
services:
- name: postgres
  file: ci/services/postgres.yml
```

When the `task` finishes (successfully or otherwise), the service will be gracefully terminated by first sending a `SIGTERM`, and eventually a `SIGKILL` if the service doesn't terminate within a timeout.

### With `across` step

Since `services` just binds to `task`, you can make use of the `across` step to run tests against a matrix of dependencies.

```yaml
across:
- var: postgres_version
  values: [9, 10, 11, 12, 13]
  max_in_flight: 3
task: integration-suite
file: ci/tasks/integration.yml
params:
  POSTGRES_ADDRESS: ((.svc:postgres.address))
services:
- name: postgres
  file: ci/services/postgres.yml
  image: postgres-((.:postgres_version))
```

## Service Configuration

Services can be configured similarly to tasks, e.g.

```yaml
name: postgres
config: # or "file:"
  image_resource: # can specify a top-level "image:" instead of "image_resource:"
    type: registry-image
    source: {repository: postgres}
  inputs:
  - name: some-input
  ports:
  - name: default # optional if using default name
    number: 5432
  startup_probe: # By default, Concourse will wait for all the listed ports to be open
    run: {path: pg_isready}
    failure_threshold: 10
    period_seconds: 5
```

Services can also run by sending a message to a [Prototype], similar to the `run` step, e.g.

```yaml
name: concourse
type: docker-compose
run: up
params:
  files:
  - concourse/docker-compose.yml
  - ci/overrides/docker-compose.ci-containerd.yml
inputs: [concourse, ci]
ports:
- name: web
  number: 8080
```

### Startup Probe

To ensure a service is ready to accept traffic before running the dependent step, the `startup_probe` must first succeed.

`startup_probe.run` defines a process to run on the service container until it succeeds. The process will run every `startup_probe.period_seconds`, and if it fails `startup_probe.failure_threshold` times, the service will error and the dependent step will not run.

If `startup_probe.run` is left unspecified, Concourse will wait for each of the specified ports to be open.

## Worker Placement

Since `services` are just bound to `task`s, the easiest approach would be to assign the service container and the task container to the same worker. This allows us to avoid a more complex architecture having to route traffic through the TSA (since workers may not be directly reachable from one another).

This hopefully isn't *too* restrictive, as anyone running e.g. `docker-compose` in a `task` for integration testing is effectively doing the same thing (just in one mega-container instead of 2+). It's also worth noting that with a native [Kubernetes Runtime], a single "worker" will likely correspond with an entire cluster, rather than a single node in the cluster.

However, it does mean that we can't provide services to tasks running on Windows/Darwin workers - not sure if there's much need for this, though.

## Networking

The way we accomplish intra-worker container-to-container networking depends on the runtime.

### Guardian and containerd

Here, we have a couple options.

1. Make use of the [Garden `NetIn` spec] to map a container port to a host port
2. Set up custom firewall rules to route traffic to containers

I'm in favour of option 2., since it doesn't require exposing ports on the worker itself. It also lets us restrict network access to the service, meaning that only the `task` container can communicate with the `service` container.

We could add a Service Manager component to each worker that the ATC will communicate with to register/unregister services. Register requests will indicate the `service` container, the exposed ports, and the `task` container that can access the `service`, while unregister requests just need to indicate the `service` container. In response, the Service Manager would create/destroy firewall rules.

![Service Manager overview](./service-manager.png)

### Kubernetes

When we build a [Kubernetes Runtime], exposing services will be much easier - we just need to create a Kubernetes `Service` (of type `ClusterIP`) exposing the service pod and appropriate ports, and `((.svc:my-service.address))` would resolve to the CoreDNS service address.

# Open Questions

* Are there (sufficiently many) practical use-cases for exposing a service to multiple steps? Or is a single `task` always sufficient?
    * psst - if you're curious about an alternative approach that allows providing services to multiple steps, and hence requires inter-worker communication, check out this [early draft] of the RFC - with a hand-wavy architecture diagram and all!
* Are there (sufficiently many) practical use-cases for exposing a service to `tasks` on Windows/Darwin workers?

# Answered Questions

# New Implications





[Prototype]: https://github.com/concourse/rfcs/blob/master/037-prototypes/proposal.md
[Kubernetes Runtime]: https://github.com/concourse/rfcs/blob/075-k8s-runtime/075-k8s-runtime/proposal.md
[Garden `NetIn` spec]: https://github.com/cloudfoundry/garden/blob/b404ff2d61e689c6510593cf75d39dc5311be663/container.go#L56-L71
[early draft]: https://github.com/aoldershaw/rfcs/blob/dc4d0082cb0441a234a775d1620ea5ed9a52a6b6/083-services/proposal.md
