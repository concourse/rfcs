# Resource Prototypes

A *resource prototype* is an interface implemented by a prototype which allows
it to be used as a *resource* in a pipeline.


## Motivation

* Support for creating multiple versions from `put`: [concourse/concourse#2660](https://github.com/concourse/concourse/issues/2660)

* Support for deleting versions: [concourse/concourse#362](https://github.com/concourse/concourse/issues/362), [concourse/concourse#524](https://github.com/concourse/concourse/issues/524)

* Make the `get` after `put` opt-in: [concourse/concourse#3299](https://github.com/concourse/concourse/issues/3299), [concourse/registry-image-resource#16](https://github.com/concourse/registry-image-resource/issues/16)


## Proposal

A resource prototype implements the following messages:

* `check`
* `get`
* `put` (optional)
* `delete` (optional)

Their behavior is described as follows.

### `check`: discover new versions of a resource

Initially, the `check` handler will be invoked with an object containing only
the fields configured by the resource's `source:` in the pipeline. A
`MessageResponse` must be emitted for *all* versions available in the source,
in chronological order.

Subsequent `check` calls will be run with a clone of the configuration object
with the last emitted version object merged in. The `check` handler must emit
the specified version if it still exists, followed by all versions detected
after it in chronological order.

If the specified version is no longer present, the `check` handler must instead
emit all available versions, as if the version was not specified. Concourse
will detect this scenario by noticing that the first version emitted does not
match the requested version. The given version, along with any other versions
that are not present in the returned set of versions, will be marked as deleted
and no longer be available.

The `check` handler can use the **bits** directory to cache state between runs.
On the first run, the directory will be empty.

There is no `check` step syntax. Concourse will `check` every configured
resource and maintain its version history.

### `get`: fetch a version of a resource

The `get` handler will be invoked with an object specifying the version to
fetch.

The `get` handler must fetch the resource into an output named `resource` under
the **bits** directory.

A `MessageResponse` must be emitted for all versions that have been fetched
into the bits directory. Each version will be recorded as an input to the
build.

When a `get` handler is invoked by a `get` step in a build plan, the output
will be mapped to the resource's name in the pipeline:

```yaml
resources:
- name: some-resource
  type: git
  source:
    uri: https://example.com/some-repo

jobs:
- name: get-and-test
  plan:
  - get: some-resource
  - task: unit
    file: some-resource/ci/unit.yml
```

Assuming the latest commit of `some-resource` is `abcdef`, this would be
similar to the following build plan:

```yaml
jobs:
- name: get-and-test
  plan:
  - run: get
    type: git
    params:
      uri: https://example.com/some-repo
      ref: abcdef
    output_mapping:
      resource: some-resource
  - task: unit
    file: some-resource/ci/unit.yml
```

The bits fetched by a `get` step may be cached so that the same version does
not have to be fetched repeatedly.


### `put`: idempotently create resource versions

The `put` handler will be invoked with user-provided configuration and
arbitrary bits.

A `MessageResponse` must be emitted for all versions that have been created/updated.

When a `put` step is used in a build plan, each version emitted will be
recorded as an output of the build.

A `get` field may be explicitly added to the step, specifying a name to fetch
the last-emitted version as:

```yaml
jobs:
- name: push-pull-resource
  plan:
  # put to 'my-resource', and then get the last emitted version object as
  # 'some-name' for use later in the build plan
  - put: my-resource
    get: some-name
```

This replaces the "implicit `get` after `put`" behavior, which will remain for
resources provided by `resource_types:` for backwards-compatibility.


### `delete`: idempotently destroy resource versions

The `delete` handler will be invoked with user-provided configuration and
arbitrary bits.

A `MessageResponse` must be emitted for all versions that have been destroyed.

When a `delete` step is used in a build plan, each version emitted will be
marked as "deleted" and no longer be available for use in other builds.

```yaml
jobs:
- name: prune-release-candidates
  plan:
  - delete: concourse-rc
    params: {regexp: concourse-[0-9]+\.[0-9]+\.[0-9]+-rc.[0-9]+.tgz}
```

## Migration Path

Concourse pipelines will support `resource_types:` and `prototypes:`
side-by-side.

Resources backed by a resource type will retain all of today's behavior for
backwards-compatibility. We don't want to force a big-bang migration of every
Concourse user's pipelines.

Resources backed by a prototype will gain all the behavior described in this
proposal. Pipeline authors are encouraged to transition from resource types to prototypes, which they can do gradually.

There is no EOL date specified for resource types support. They would likely be
supported for a number of years.

## Open Questions

n/a

## Answered Questions

* [Version filtering is best left to `config`.](https://github.com/concourse/concourse/issues/1176#issuecomment-472111623)
* [Resource-determined triggerability of versions](https://github.com/concourse/rfcs/issues/11) will be addressed by the "trigger resource" RFC.
* Webhooks are left out of this first pass of the interface. I would like to investigate alternative approaches before baking it in.
  * For example, could Concourse itself integrate with these services and map webhooks to resource checks intelligently?
