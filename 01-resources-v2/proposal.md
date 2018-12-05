# Summary

This RFC proposes a new resource interface to replace the existing resource
interface.

As part of this proposal, the interface will now be versioned, starting at 2.0.
Today's resource interface (documented
[here](https://github.com/concourse/docs/blob/b9d291e5a821046b8a5de48c50b5ccba5a977493/lit/reference/resource-types/implementing.lit))
will be called version 1, even though it was never really versioned.

The introduction of this new interface will be gradual, allowing Concourse
users to use a mix of v1 and v2 resources throughout their pipelines. While the
new interface is defined in terms of entirely new concepts like
[spaces](https://github.com/concourse/concourse/issues/1707), v1 resources will
be silently 'adapted' to v2 automatically.


# Motivation

* Support for multi-branch workflows and build matrixes:
    * https://github.com/concourse/concourse/issues/1172
    * https://github.com/concourse/concourse/issues/1707

* Support for creating new branches dynamically (as spaces):
    * https://github.com/concourse/git-resource/pull/172

* Support for creating multiple versions at once:
    * https://github.com/concourse/concourse/issues/535
    * https://github.com/concourse/concourse/issues/2660

* Support for deleting versions:
    * https://github.com/concourse/concourse/issues/362

* Having resource metadata immediately available via `check`:
    * https://github.com/concourse/git-resource/issues/193

* Unifying `source` and `params` as just `config` so that resources don't have
  to care where configuration is being set in pipelines:
    * https://github.com/concourse/concourse/issues/310

* Improving stability of reattaching to builds by reading resource responses
  from files instead of `stdout`:
    * https://github.com/concourse/concourse/issues/1580

* Ensuring resource version history is always correct and up-to-date, enabling
  it to be [deduped](https://github.com/concourse/concourse/issues/2386) and
  removing the need for [purging
  history](https://github.com/concourse/concourse/issues/145) and
  [removing/renaming
  resources](https://github.com/concourse/concourse/issues/372).

* Closing gaps in the resource interface that turned them into a "local maxima"
  and resulted in their being used in somewhat cumbersome ways (notifications,
  partially-implemented resources, etc.)


# Proposal

* TODO: document 'info'
* TODO: make metadata more flexible?
    * https://github.com/concourse/concourse/issues/310
    * https://github.com/concourse/concourse/issues/2900


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

The `check` script responds by writing JSON objects ("events") to a file
specified by `response_path`. Each JSON object has an `action` and a different
set of fields based on the action.

The following event types may be emitted by `check`:

* `default_space`: Emitted when the resource has learned of a space which
  should be considered the "default", e.g. the default branch of a `git` repo
  or the latest version available for a semver'd resource.

  Required fields for this event:

  * `space`: The name of the space.

* `discovered`: Emitted when a version is discovered for a given space. These
  must be emitted in chronological order (relative to other `discovered` events
  for the given space - other events may be intermixed).

  Required fields for this event:

  * `space`: The space the version is in.
  * `version`: The version object.
  * `metadata`: A list of JSON objects with `name` and `value`, shown to the
    user.

* `reset`: Emitted when a given space's "current version" is no longer present
  (e.g. someone ran `git push -f`). This has the effect of marking all
  currently-recorded versions of the space 'deleted', after which the resource
  will emit any and all versions from the beginning, thus 'un-deleting'
  anything that's actually still there.

  Required fields for this event:

  * `space`: The name of the space.

The first request will have an empty object as `from`.

Any spaces discovered by the resource but not present in `from` should emit
versions from the very first version.

For each space and associated version in `from`, the resource should emit all
versions that appear *after* the given version (not including the given
version).

If a space or given version in `from` is no longer present (in the case of `git
push -f` or branch deletion), the resource should emit a `reset` event for the
space. If the space is still there, but the verion was gone, it should follow
the `reset` event with all versions detected from the beginning, as if the
`from` value was never specified.

The resource should determine a "default space", if any. Having a default space
is useful for things like Git repos which have a default branch, or version
spaces (e.g. `1.8`, `2.0`) which can point to the latest version line by
default. If there is no default space, the user must specify it explicitly in
the pipeline, either by configuring one on the resource (`default_space: foo`)
or on every `get` step using the resource (`spaces: [foo]`).

#### example

Given the following request on `stdin`:

```json
{
  "config": {
    "uri": "https://github.com/concourse/concourse"
  },
  "from": {
    "master": {"ref": "abc123"},
    "feature/foo": {"ref":"def456"},
    "feature/bar": {"ref":"987cia"}
  },
  "response_path": "/tmp/check-response.json"
}
```

If the `feature/foo` branch has new commits, `master` is the default branch and
has no new commits, and `feature/bar` has been `push -f`ed, you may see
something like the following in `/tmp/check-response.json`:

```json
{"action":"discovered","space":"feature/foo","version":{"ref":"abcdf8"},"metadata":[{"name":"message","value":"fix thing"}]}
{"action":"reset","space":"feature/bar"}
{"action":"discovered","space":"feature/bar","version":{"ref":"abcde0"},"metadata":[{"name":"message","value":"initial commit"}]}
{"action":"discovered","space":"feature/bar","version":{"ref":"abcde1"},"metadata":[{"name":"message","value":"add readme"}]}
{"action":"default_space","space":"master"}
{"action":"discovered","space":"feature/foo","version":{"ref":"abcdf9"},"metadata":[{"name":"message","value":"fix thing even more"}]}
{"action":"discovered","space":"feature/bar","version":{"ref":"abcde2"},"metadata":[{"name":"message","value":"finish the feature"}]}
```

A few things to note:

* A `reset` event is emitted immediately upon detecting that the given version
  for `feature/bar` (`987cia`) is no longer available, followed by a
  `discovered` event for every commit going back to the initial commit on the
  branch.

* No versions are emitted for `master`, because it's already up to date
  (`abc123` is the latest commit).

* The versions detected for `feature/foo` may appear between events for
  `feature/bar`, as they're for unrelated spaces. The order only matters within
  the space.


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

The `put` script responds by writing JSON objects ("events") to a file
specified by `response_path`, just like `check`. Each JSON object has an
`action` and a different set of fields based on the action.

Anything printed to `stdout` and `stderr` will propagate to the build logs.

The following event types may be emitted by `put`:

* `created`: Emitted when the resource has created (perhaps idempotently) a
  version. The version will be recorded as an output of the build.

  Versions produced by `put` will *not* be directly inserted into the
  resource's version history in the pipeline, as they were with v1 resources.
  This enables one-off versions to be created and fetched within a build
  without disrupting the normal detection of resource versions across the

  Required fields for this event:

  * `space`: The space the version is in.
  * `version`: The version object.
  * `metadata`: A list of JSON objects with `name` and `value`, shown to the
    user. Note that this is return by both `put` and `check`, because there's a
    chance that `put` produces a version that wouldn't normally be discovered
    by `check`.

* `deleted`: Emitted when a version has been deleted. The version record will
  remain in the database for archival purposes, but it will no longer be a
  candidate for any builds.

  Required fields for this event:

  * `space`: The space the version is in.
  * `version`: The version object.

Because the space is included on each event, `put` allows a new space to be
generated dynamically (based on params and/or the bits in its working
directory) and propagated to the rest of the pipeline. However it must take
care to only affect one space at a time. Without this restriction it becomes
difficult to express things like "`get` after `put`" to fetch the version that
was created. If multiple spaces are returned, it's unclear which space the
`get` would fetch from.

#### the `get` after the `put`

With v1 resources, every `put` implied a `get` of the version that was created.
With v2 we will change that, so that the `get` is opt-in. This has been a
long-time ask, and one objective reason to make it opt-in is that Concourse
can't know ahead of time that there will even be anything to `get` - for
example, the `put` could emit only `deleted` events.

So, to `get` the latest version that was produced by the `put`, you would
configure something like:

```yaml
- put: my-resource
  get: my-created-resource
- task: use-my-created-resource
```

The value for the `get` field is the name of the artifact to save. When
specified, the last version emitted will be fetched.


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
  easier to implement a resource without planning ahead for dynamic vs. static
  usage patterns. This will become more powerful if concourse/concourse#684 is
  implemented.

* Change `check` to run against all spaces. It will be given a mapping of each
  space to its current latest version, and return the set of all spaces, along
  with any new versions in each space.

  This is all done as one batch call so that resources can decide how to
  efficiently perform the check. It also keeps the container overhead down to
  one per resource, rather than one per space.

* Remove the implicit `get` after every `put`, now requiring the pipeline to
  explicitly configure a `get` field on the same step. This is necessary now
  that `put` can potentially perform an operation resulting solely in `deleted`
  events, in which case there is nothing to fetch.

  This has also been requested by users for quite a while, for the sake of
  optimizing jobs that have no need for the implicit `get`.

* Change `put` to emit a sequence of created versions, rather than just one.

  Technically the `git` resource may push many commits, so returning more than
  one version is necessary to track them all as outputs of a build. This could
  also support batch creation.

  To ensure `check` is the source of truth for ordering, the versions emitted
  by `put` are not directly inserted into the database. Instead, they are
  simply recorded as outputs of the build. The order does matter, however - if
  a user configures a `get` on the `put` step, the last version emitted will be
  fetched. For this reason they should be emitted in chronological order.

* Change `put` to additionally return a sequence of *deleted* versions.

  There has long been a call for a batch `delete` or `destroy` action. Adding
  this to `put` alongside the set of created versions allows `put` to become a
  general idempotent side-effect performer, rather than implying that each
  resource must support a separate `delete` action.

* Change `get` to always run against a particular space, given by
  the request payload.

* Change `check` to include metadata for each version. Change `get` to no
  longer return it.

  This way metadata is always immediately available, which could enable us to
  have a richer UI for the version history page.

  The original thought was that metadata collection may be expensive, but so
  far we haven't seen that to be the case.

* Change `get` script to no longer return a version, since it's always given
  one now. As a result, `get` no longer has a response; it just succeeds or
  fails.

* Change `get` and `put` to run with the bits as their working directory,
  rather than taking the path as an argument. This was something people would
  trip up on when implementing a resource.

* Change `check` and `put` to write its JSON response to a specified file,
  rather than `stdout`, so that we don't have to be attached to process its
  response.

  This is one of the few ways a build can error after the ATC reattaches
  (`unexpected end of JSON`). With it written to a file, we can just try to
  read the file when we re-attach after seeing that the process exited. This
  also frees up `stdout`/`stderr` for normal logging, which has been an
  occasional pitfall during resource development/debugging.

  Another motivation for this is safety: with `check` emitting a ton of data,
  there is danger in Garden losing windows of the output due to a slow
  consumer. Writing to a file circumvents this issue.


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

## Build-local Versions

Now that `put` doesn't directly modify the resource's version history, it can
be used to provide explicitly versioned 'variants' of original versions without
doubling up the version history. One use case for this is pull-requests: you
may want a build to pull in one resource for the PR itself, another resource
for the base branch of the upstream reap, and then `put` to produce a
"combined" version of the two, representing the PR merged into the upstream
repo:

```yaml
jobs:
- name: run-pr
  plan:
  - get: concourse-pr  # pr: 123, ref: deadbeef
    trigger: true
  - get: concourse     # ref: abcdef
  - put: concourse-pr
    get: merged-pr
    params:
      merge_base: concourse
      status: pending

    # the `put` will learns base ref from `concourse` input and param, and emit
    # a 'created' event with the following version:
    #
    #   pr: 123, ref: deadbeef, base: abcdef
    #
    # the `get` will then run with that version and knows to merge onto the
    # given base ref

  - task: unit
    # uses 'merged-pr' as an input
```

# Implementation Notes

## Performance Implications

Now that we're going to be collecting all versions of every resource, we should
be careful not to be scanning the entire table all the time, and even make an
effort to share data when possible. We have implemented this with
https://github.com/concourse/concourse/issues/2386.
