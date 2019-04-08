# Artifact Resources

This proposal uses the [generalized resource interface](../024-generalized-resources/proposal.md) to show how the interface would be implemented and interpreted by Concourse to support "artifact resources", today's primary use case for v1 resources.

## Motivation

* Support for creating multiple versions from `put`: [concourse/concourse#2660](https://github.com/concourse/concourse/issues/2660)

* Support for deleting versions: [concourse/concourse#362](https://github.com/concourse/concourse/issues/362), [concourse/concourse#524](https://github.com/concourse/concourse/issues/524)

* Make the `get` after `put` opt-in: [concourse/concourse#3299](https://github.com/concourse/concourse/issues/3299), [concourse/registry-image-resource#16](https://github.com/concourse/registry-image-resource/issues/16)

## Examples

* A basic implementation of the `git` resource can be found in [`git-v2`](examples/git-v2).

## Proposal

A v2 resource type can be used as a versioned artifact resource by treating the **config fragments** as **versions** and emitting them in chronological order from `check`. This way the resource type is used to model change in an external resource over time.

### `check`: discover versions in order

The `check` action will first be run with a "naked" config, containing only what the user specified. In this situation `check` must emit an `ActionResponse` for all versions discovered in the config, in chronological order.

Subsequent calls to `check` will be given a config that has been spliced with the last emitted version config fragment. The `check` script must an `ActionResponse` for the given version if it still exists, followed by a response for any versions that came after it.

If the specified version is no longer present, the `check` action must go back to returning all versions, as if the version was not specified in the first place. Concourse will detect this scenario by noticing that the first `ActionResponse` emitted does not match the requested version. All versions that existed before that were emitted will be automatically marked "deleted".

The `check` action can use the **bits** directory to cache state between runs of the `check` on that worker. On the first run, the directory will be empty.

### `get`: fetch a version of an artifact

The `get` action will always be invoked with a spliced config specifying which version to fetch. It is given an empty **bits** directory in which to fetch the data.

An `ActionResponse` must be emitted for all versions that have been fetched into the bits directory. Each version will be recorded as an input to the build.

### `put`: idempotently create artifact versions

The `put` action will be invoked with user-provided configuration and arbitrary bits.

An `ActionResponse` must be emitted for all versions that have been created/updated. Each version will be recorded as an output of the build.

### `delete`: idempotently destroy artifact versions

The `delete` action will be invoked with user-provided configuration and arbitrary bits.

An `ActionResponse` must be emitted for all versions that have been destroyed. These versions will be marked "deleted" and no longer be available for use in other builds.

## Open Questions

* [resource-determined triggerability of versions?](https://github.com/concourse/rfcs/issues/11)
* webhooks?
  * should these instead be something supported by *Concourse*?

## Answered Questions

* Version filtering is probably best left to `config`.