# Artifact Resources

This proposal uses the [generalized resource interface](../024-generalized-resources/proposal.md) to show how the interface would be implemented and interpreted by Concourse to support "artifact resources", today's primary use case for v1 resources.

## Motivation

* Support for creating multiple versions from `put`: [concourse/concourse#2660](https://github.com/concourse/concourse/issues/2660)

* Support for deleting versions: [concourse/concourse#362](https://github.com/concourse/concourse/issues/362), [concourse/concourse#524](https://github.com/concourse/concourse/issues/524)

* Make the `get` after `put` opt-in: [concourse/concourse#3299](https://github.com/concourse/concourse/issues/3299), [concourse/registry-image-resource#16](https://github.com/concourse/registry-image-resource/issues/16)

## Examples

A basic implementation of the `git` resource can be found in [`git-v2`](examples/git-v2).

## Proposal

All resources as of Concourse v5.2 are technically artifact resources. They are now explicitly named artifact resources to disambiguate from other interpretations of the resource interface (e.g. spatial resources, notification resources, trigger resources).

This proposal describes how the new resource interface will be interpreted in order to support artifacts, and also proposes pipeline changes in order to support a backwards-compatible transition to the now-explicit "`get` after `put`" semantics.

### Resource interface v2 interpretation

A v2 resource type can be used as an artifact resource by treating the **config fragments** as **versions** and emitting them in chronological order from `check`. This way the resource type is used to model change in an external resource over time.

#### `check`: discover versions in order

The `check` action will first be run with a "naked" config, containing only what the user specified. In this situation `check` must emit an `ActionResponse` for all versions discovered in the config, in chronological order.

Subsequent calls to `check` will be given a config that has been spliced with the last emitted version config fragment. The `check` script must an `ActionResponse` for the given version if it still exists, followed by a response for any versions that came after it.

If the specified version is no longer present, the `check` action must go back to returning all versions, as if the version was not specified in the first place. Concourse will detect this scenario by noticing that the first `ActionResponse` emitted does not match the requested version. All versions that existed before that were emitted will be automatically marked "deleted".

The `check` action can use the **bits** directory to cache state between runs of the `check` on that worker. On the first run, the directory will be empty.

#### `get`: fetch a version of an artifact

The `get` action will always be invoked with a spliced config specifying which version to fetch. It is given an empty **bits** directory in which to fetch the data.

An `ActionResponse` must be emitted for all versions that have been fetched into the bits directory. Each version will be recorded as an input to the build.

#### `put`: idempotently create artifact versions

The `put` action will be invoked with user-provided configuration and arbitrary bits.

An `ActionResponse` must be emitted for all versions that have been created/updated. Each version will be recorded as an output of the build.

#### `delete`: idempotently destroy artifact versions

The `delete` action will be invoked with user-provided configuration and arbitrary bits.

An `ActionResponse` must be emitted for all versions that have been destroyed. These versions will be marked "deleted" and no longer be available for use in other builds.

### Making `get` after `put` explicit

Artifact resources will be defined at the top-level in the pipeline, under a new field called `artifacts:`. This field replaces `resources:` and has exactly the same structure.

Example:

```yaml
artifacts: # instead of 'resources:'
- name: booklit
  type: git
  source: {uri: "https://github.com/vito/booklit"}

jobs:
- name: unit
  plan:
  - get: booklit
    trigger: true
  - task: test
    file: booklit/ci/test.yml
```

When `artifacts:` is defined instead of `resources:`, `put` steps will no longer imply an automatic `get` step. Instead, a `get` field must be explicitly added to the step:

```yaml
artifacts:
- name: my-resource
  type: git
  source: # ...

jobs:
- name: push-pull-resource
  plan:
  - put: my-resource
    get: input-name
  # the created artifact will be fetched as 'input-name'
```

This change will be backwards-compatible - `resources:` will be treated as `artifacts:` and continue to use the "automatic `get` after `put`" behavior. Configuring pipelines with `resources:` will result in a deprecation warning instructing them to switch to `artifacts:` and migrate to the new "explicit `get` after `put`" behavior.

## Open Questions

n/a

## Answered Questions

* [Version filtering is best left to `config`.](https://github.com/concourse/concourse/issues/1176#issuecomment-472111623)
* [Resource-determined triggerability of versions](https://github.com/concourse/rfcs/issues/11) will be addressed by the "trigger resource" RFC.
* Webhooks are left out of this first pass of the interface. I would like to investigate alternative approaches before baking it in.
  * For example, could Concourse itself integrate with these services and map webhooks to resource checks intelligently?