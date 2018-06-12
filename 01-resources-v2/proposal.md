# Summary

Introduces a new resource interface with the following goals:

* Introduce versioning to the resource interface, so that we can maintain
  backwards-compatibility.

* Support for spaces (concourse/concourse#1707).

* Introduce a more airtight "versioned artifacts" interface, tightening up
  loopholes in today's resource API to ensure that resources are pointing at an
  external source of truth and cannot be partially implemented or hacky.

* Extend the versioned artifacts interface to support deletion of versions,
  either by an explicit `delete` call or by somehow noticing that versions have
  disappeared.


# Proposal

At this early stage of the RFC, it's easiest for me to just use Elm syntax and
pretend this is all in a type system.

In reality, each of these functions would be scripts with JSON requests passed
to them on stdin. I'm going to use the `Bits` type just to represent which calls
have state on disk to either access (like `put`) or return (like `get`). This
would normally just be the working directory of the script.

## General Types

```elm
-- arbitrary configuration
type alias Config = Dict String Json.Value

-- identifier for space, i.e. 'foo' or '1.2'
type alias Space = String

-- identifier for version, i.e. {"version":"1.2"}
type alias Version = Dict String String

-- arbitrary ordered metadata (we may make this fancier in the future)
type alias Metadata = List (String, String)

-- data on disk
type alias Bits = ()
```

## Versioned Artifacts interface

```elm
check   : Config -> Dict Space Version -> Dict Space (List (Version, Metadata))
get     : Config -> Space -> Version -> Bits
put     : Config -> Bits -> Dict Space { created : Set Version, deleted : Set Version }
```

# Examples

## Resource Implementations

I've started implementing a new `git` resource alongside this
document. See
[`git-example/`](https://github.com/vito/rfcs/tree/resources-v2/01-resources-v2/git-example).
I've left `TODO`s for parts that need more thinking or discussion. Please
leave comments!

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

* Change `put` to emit a set of created versions for each space, rather than
  just one.

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
~~With spaces there will be more `check`s than ever. Right now, there's one
container per recurring `check`. Can we reduce the container overhead here by
requiring that resource `check`s be side-effect free and able to run in
parallel?~~
</p>

<p>
~~There may be substantial security implications for this.~~
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
~~It may be the case that most resources cannot easily support `destroy`. One
example is the `git` resource. It doesn't really make sense to `destroy` a
commit. Even if it did (`push -f`?), it's a kind of weird workflow to support
out of the box.~~
</p>

<p>
~~Could we instead just have `put` and ensure that we `check` in such a way that
deleted versions are automatically noticed? What would the overhead of this
be?~~ This only works if the versions are "chained", as with the `git` case.
</p>

<p>
Decided against introducing `destroy` in favor of having `put` return two sets
for each space: versions created and versions deleted. This generalizes `put`
into an idempotent versioned artifact side effect performer.
</p>
</details>

<details><summary>Should `put` be given a space or return the space?</summary>

<p>
~~The verb `PUT` in HTTP implies an idempotent action against a given resource. So
it's intuitive that the `put` verb here would do the same.~~
</p>
<p>
~~However, many of today's usage of `put` would be against a dynamically
determined space. For example, most semver workflows involve `put`ing with the
version determined by a file (often coming from the `semver` resource). So the
space isn't known statically at pipeline configuration time.~~
</p>
<p>
~~What's more, the resulting space for a semver push would only be `MAJOR.MINOR`,
excluding the final patch segment. This is annoying to have to explicitly
configure in your build.~~
</p>
<p>
~~If we instead have `put` return both the space and the versions, this would be a
lot simpler.~~
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
