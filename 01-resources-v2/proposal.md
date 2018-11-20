# Summary

Introduces a new resource interface with the following goals:

* Introduce versioning to the resource interface, so that we can maintain
  backwards-compatibility.

* Support for spaces (concourse/concourse#1707).

* Introduce a more airtight "versioned artifacts" interface, tightening up
  loopholes in today's resource API to ensure that resources are pointing at an
  external source of truth, so that we can explicitly design for new features
  and workflows rather than forcing everything into the resource interface.

* Extend the versioned artifacts interface to support deletion of versions.


# Proposal

## General Types

```go
// Space is a name of a space, e.g. "master", "release/3.14", "1.0".
type Space string

// Config is a black box containing all user-supplied configuration, combining
// `source` in the resource definition with `params` from the step (in the
// case of `get` or `put`).
type Config map[string]interface{}

// Version is a key-value identifier for a version of a resource, e.g.
// `{"ref":"abcdef"}`, `{"version":"1.2.3"}`.
type Version map[string]string

// Metadata is an ordered list of metadata fields to display to the user about
// a resource version. It's ordered so that the resource can decide the best
// way to show it.
type Metadata []MetadataField

// MetadataField is an arbitrary key-value to display to the user about a
// version of a resource.
type MetadataField struct {
  Name  string `json:"name"`
  Value string `json:"value"`
}
```

## Versioned Artifacts interface

### `check`: Detect versions across spaces.

The `check` command will be invoked with the following JSON structure on
`stdin`:

```go
// CheckRequest contains the resource's configuration and latest version
// associated to each space.
type CheckRequest struct {
  Config       Config            `json:"config"`
  From         map[Space]Version `json:"from"`
  ResponsePath string            `json:"response_path"`
}
```

The first call will have an empty object as `from`.

Any spaces discovered by the resource but not present in `from` should collect
versions from the beginning.

For each space in `from`, the resource should collect all versions that appear
*after* the current version, including the given version if it's still present.
If the given version is no longer present, the resource should instead collect
from the beginning, as if the space was not specified.

If any space in `from` is no longer present, the resource should ignore it, and
not include it in the response.

The resource should determine a "default space", if any. Having a default
space is useful for things like Git repos which have a default branch, or
version spaces (e.g. `1.8`, `2.0`) which can point to the latest version line by
default. If there is no default space, the user must specify it explicitly in
the pipeline, either by configuring one on the resource (`space: foo`) or on the
`get` step (`spaces: [foo]`).

The command should first emit the default space (if any) and then stream the
collected versions for each space. Streaming would enable resource authors to
write versions as they find them rather than hold them all in memory and do one
big JSON marshal. Each version will be written as individual JSON objects
streamed to the response_path file. They will include the version, the space
associated with that version, and the version's metadata. The response should
look like the following within the file specified by response_path:

```JSON
{"space":"a","version":{"v":"1"},"metadata":[{"Name":"status","Value":"pending"}]}
{"space":"a","version":{"v":"2"},"metadata":[{"Name":"status","Value":"pending"}]}
{"space":"a","version":{"v":"3"},"metadata":[{"Name":"status","Value":"pending"}]}
{"space":"b","version":{"v":"1"},"metadata":[{"Name":"status","Value":"pending"}]}
{"space":"b","version":{"v":"2"},"metadata":[{"Name":"status","Value":"pending"}]}
{"space":"b","version":{"v":"3"},"metadata":[{"Name":"status","Value":"pending"}]}
// ...
```

### `get`: Fetch a version from the resource's space.

The `get` command will be invoked with the following JSON structure on `stdin`:

```go
type GetRequest struct {
  Config  Config  `json:"config"`
  Space   Space   `json:"space"`
  Version Version `json:"version"`
}
```

The command will be invoked with a completely empty working directory. The
command should populate this directory with the requested bits. The `git`
resource, for example, would clone directly into the working directory.

If the requested version is unavailable, the command should exit nonzero.

No response is expected.

Anything printed to `stdout` and `stderr` will propagate to the build logs.


### `put`: Idempotently create or destroy resource versions in a space.

The `put` command will be invoked with the following JSON structure on `stdin`:

```go
type PutRequest struct {
  Config       Config `json:"config"`
  ResponsePath string `json:"response_path"`
}
```

The command will be invoked with all of the build plan's artifacts present in
the working directory, each as `./(artifact name)`.

The command should perform any and all side-effects idempotently, and then
stream the following response to the file specified by `response_path`:

```JSON
{"type":"created","space":"a","version":{"v":"1"}}
{"type":"created","space":"a","version":{"v":"2"}}
{"type":"created","space":"a","version":{"v":"3"}}
{"type":"created","space":"b","version":{"v":"1"}}
{"type":"created","space":"b","version":{"v":"2"}}
{"type":"deleted","space":"b","version":{"v":"3"}}
// ...
```

`put` allows new spaces to be generated dynamically (based on params
and/or the bits in its working directory) and propagated to the rest of the
pipeline.

Note that a `put` may only affect one space at a time, otherwise it becomes
difficult to express things like "`get` after `put`" to fetch the version that
was created. If multiple spaces are returned, it's unclear which space the
`get` would fetch from.

Versions returned with `created` type will be recorded as outputs of the build.
A `check` will then be performed to fill in the metadata and determine the
ordering of the versions. Once the ordering is learned, the latest version will
be fetched by the implicit `get`.

Versions returned with `deleted` type will be marked as deleted. They will
remain in the database for archival purposes, but will no longer be input
candidates for any builds, and can no longer be fetched. The implicit `get`
after `deleted` puts should not happen so this will result in changes to that
behavior.

Anything printed to `stdout` and `stderr` will propagate to the build logs.


# Examples

## Resource Implementations

I've started cooking up new resources using this interface. I've left `TODO`s
for parts that need more thinking or discussion. Please leave comments!

### `git`

[Code](https://github.com/vito/rfcs/tree/resources-v2/01-resources-v2/git-example)

This resource models the original `git` resource. It represents each branch as a space.

### `semver-git`

[Code](https://github.com/vito/rfcs/tree/resources-v2/01-resources-v2/semver-example)

This is a whole new semver resource intended to replace the original `semver`
resource with a better model that supports concurrent version lines (i.e.
supporting multiple major/minor releases with patches). It does this by managing
tags in an existing Git repository.

### `s3`

[Code](https://github.com/vito/rfcs/tree/resources-v2/01-resources-v2/s3-example)

This resource models the original `s3` resource. Only regex versions were
implemented, each space corresponds to a major.minor version. For example, 1.2.0
and 1.2.1 is the same space but 1.3.0 is a different space. Single numbers are
also supported with default minor of 0. The default space is set to the latest
minor version.


## Pipeline Usage

TODO:

- Pull Requests
- Feature branches
- Build matrixes
- Generating branches (and propagating them downstream)
- Semver artifacts
- Fanning out against multiple IaaSes
- Pool resource?
- BOSH deploys


# Summary of Changes

## Overarching Changes

* Add an `info` script which returns a JSON object indicating the supported
  interfaces, their protocol versions, and any other interface-specific
  meta-configuration (for example, which commands to execute for the
  interface's hooks).

* The first supported interface will be called `artifacts`, and its version
  will start at `2.0` as it's really the next iteration of the existing
  "resources" concept, but with a more specific name.

* There are no more hardcoded paths (`/opt/resource/X`) - instead there's the
  single `info` entrypoint, which is run in the container's working directory.
  This is more platform-agnostic.


## Changes to Versioned Artifact resources

* Remove the distinction between `source` and `params`; resources will receive
  a single `config`. The distinction will remain in the pipeline. This makes it
  easier to implement a resource without planning ahead for interesting dynamic
  vs. static usage patterns, and this will become more powerful with
  concourse/concourse#684.

* Change `check` to run against all spaces. It will be given a mapping of each
  space to its current latest version, and return the set of all spaces, along
  with any new versions in each space.

  This is all done as one batch call so that resources can decide how to
  efficiently perform the check. It also keeps the container overhead down to
  one per resource, rather than one per space.

* Change `put` to emit a set of created versions, rather than just one.

  Technically the `git` resource may push many commits, so returning more than
  one version is necessary to track them all as outputs of a build. This could
  also support batch creation.

  To ensure `check` is the source of truth for ordering, the versions returned
  by `put` are not order dependent. A `check` will be performed to discover
  them in the correct order, and then each version will be saved as an output
  of the build. The latest version of the set will then be fetched.

* Change `put` to additionally return a set of *deleted* versions.

  There has long been a call for a batch `delete` or `destroy` action. Adding
  this to `put` alongside the set of created versions allows `put` to become a
  general idempotent side-effect performer, rather than implying that each
  resource must support a separate `delete` action.

* Change `get` to always run against a particular space, given by
  the request payload.

* Change `check` to include metadata for each version. Change `get` and `put`
  to no longer return it.

  This way metadata is always immediately available, and only comes from one
  place.

  The original thought was that metadata collection may be expensive, but so
  far we haven't seen that to be the case.

* Change `get` script to no longer return a version, since it's always given
  one now. As a result, `get` no longer has a response; it just succeeds or
  fails.

* Change `get` and `put` to run with the bits as their working directory,
  rather than taking the path as an argument. This was something people would
  trip up on when implementing a resource.

* Change `put` to write its JSON response to a specified file, rather than
  `stdout`, so that we don't have to be attached to process its response.

  This is one of the few ways a build can error after the ATC reattaches
  (`unexpected end of JSON`). With it written to a file, we can just try to
  read the file when we re-attach after seeing that the process exited. This
  also frees up stdout/stderr for normal logging, which has been an occasional
  pitfall during resource development/debugging.


# Answered(?) Questions

<details><summary>Can we reduce the `check` overhead?</summary>

<p>
<strike>With spaces there will be more `check`s than ever. Right now, there's one
container per recurring `check`. Can we reduce the container overhead here by
requiring that resource `check`s be side-effect free and able to run in
parallel?</strike>
</p>

<p>
<strike>There may be substantial security implications for this.</strike>
</p>

<p>
This is now done as one big `check` across all spaces, run in a single
container. Resources can choose how to perform this efficiently and safely.
This may mean GraphQL requests or just iterating over local shared state in
series. Even in the worst-case, where no parallelism is involved, it will at
least consume only one container.
</p>
</details>

<details><summary>Is `destroy` general enough to be a part of the interface?</summary>

<p>
<strike>It may be the case that most resources cannot easily support `destroy`. One
example is the `git` resource. It doesn't really make sense to `destroy` a
commit. Even if it did (`push -f`?), it's a kind of weird workflow to support
out of the box.</strike>
</p>

<p>
<strike>Could we instead just have `put` and ensure that we `check` in such a way that
deleted versions are automatically noticed? What would the overhead of this
be?</strike> This only works if the versions are "chained", as with the `git` case.
</p>

<p>
Decided against introducing `destroy` in favor of having `put` return two sets
for each space: versions created and versions deleted. This generalizes `put`
into an idempotent versioned artifact side effect performer.
</p>
</details>

<details><summary>Should `put` be given a space or return the space?</summary>

<p>
<strike>The verb `PUT` in HTTP implies an idempotent action against a given resource. So
it's intuitive that the `put` verb here would do the same.</strike>
</p>
<p>
<strike>However, many of today's usage of `put` would be against a dynamically
determined space. For example, most semver workflows involve `put`ing with the
version determined by a file (often coming from the `semver` resource). So the
space isn't known statically at pipeline configuration time.</strike>
</p>
<p>
<strike>What's more, the resulting space for a semver push would only be `MAJOR.MINOR`,
excluding the final patch segment. This is annoying to have to explicitly
configure in your build.</strike>
</p>
<p>
<strike>If we instead have `put` return both the space and the versions, this would be a
lot simpler.</strike>
</p>
<p>
Answered this at the same time as having `put` return a set of deleted
versions. It'll return multiple spaces and versions created/deleted for them.
</p>
</details>


# New Implications

Here are a few use cases that resources were sometimes used for inappropriately:

## Single-state resources

Resources that really only have a "current state", such as deployments. This is
still "change over time", but the difference is that old versions become
invalid as soon as there's a new one. This can now be made more clear by
marking the old versions as "deleted", either proactively via `put` or by
`check` discovering the new version.

## Non-linearly versioned artifact storage

This can be done by representing each non-linear version in a separate space.
For example, generated code could be pushed to a generated (but deterministic)
branch name, and that space could then be passed along.


# Implementation Notes

## Performance Implications

Now that we're going to be collecting all versions of every resource, we should
be careful not to be scanning the entire table all the time, and even make an
effort to share data when possible. For example, we may want to associate
collected versions to a global resource config object, rather than saving them
all per-pipeline-resource.

Here are some optimizations we probably want to make:

* `(db.Pipeline).GetLatestVersionedResource` is called every minute and scans
  the table to find the latest version of a given resource. We should reduce
  this to a simple join column from the resource to the latest version,
  maintained every time we save new versions.
