# Summary

Introduces a new resource interface with the following goals:

1. Support for spaces (concourse/concourse#1707).

1. Marking versions as deleted.

1. Tightening up loopholes in the API to ensure that resources are pointing at
   an external source of truth and cannot be "partially implemented".

1. Introduce versioning to the resource interface, so that we can maintain
   backwards-compatibility.

1. Establish a pattern for notification-style resources like Slack, GitHub
   commit/PR status, etc.


# Motivation

Resources today are used for many things that they should not be used for. This
leads to surprising behavior, difficult workarounds, and a lack of consistency
in what you can expect from any given resource type.

Many of them only implement a subset of the interface, or only provide stubs
for part of it. This makes them not a true resource.

We have a need in concourse/concourse#1707 to discover change over *space*, not
*time*, which requires additions or changes to the resource interface. We also
have


## New Implications

Here are a few use cases that resources were sometimes used for inappropriately:

1. Resources that really only have a "current state", such as deployments. This
  is still "change over time", but the difference is that old versions become
  invalid as soon as there's a new one.

1. Pushing un-versioned artifacts through a pipeline, such that the version
  history for the resource becomes nonlinear (the "latest" version has nothing
  to do with the versions prior). This is a cardinal sin.

The new interface improves the story around these two use cases.

The first case is resolved by having versions be deletable; we can now
represent that previous states of an external dependency are no longer
available.

The second use case can be resolved by using a new space to represent nonlinear
versions.


## xx hooks use case

- git: commit status
- github pr: pr status

- pool?: validate lock still available on start? configurable to release on sad path? (maybe from get/put params?)
- slack notification: `put` to start msg, hooks reply to it with result? (kind of an abuse tho)

# Proposal

```elm
-- arbitrary configuration
type alias Config = Dict String Json.Value

-- data on disk (it's not in memory, so blank return value)
type alias Bits = ()

-- arbitrary metadata to show to the user (this may become more structured later)
type alias Metadata = List (String, String)

-- Versioned Dependencies (Git repo)
type alias Version = Dict String String

type alias Space = Dict String String

discover : Config -> Set Space
check    : Config -> Space -> Maybe Version -> List Version
get      : Config -> Space -> Version -> Bits
put      : Config -> Space -> Bits -> Set Version
destroy  : Config -> Space -> Bits -> Set Version

notify : Config -> Notification -> ()

-- Optional: runs whenever build status changes while using resource as input
-- notify : Config -> Space -> Version -> Status -> ()

type alias BuildMetadata =
  { teamName : String
  , pipelineName : String
  , jobName : String
  , buildName : String
  , buildID : Int
  , status : String
  }
```


# Examples

- Pull Requests
- BOSH deploys
- Feature branches
- Generated branches
- IaaSes
- Pool resource?


## Pull Requests

!! stateful spaces? hooks on build start/finish with state available
!! this could be used to reflect PR status, send slack alerts, track pool entries
!! these hooks might not just be notifications - it could be important (ie release lock on abort)
!! need a way for `put` to create a new space


### Scheduling

There should be a single "space combination" of `atc-pr-unit` for every active
space returned by `atc-prs`. So, the `atc-prs` space must be periodically
`check`ed to determine the set of PRs available, and the `atc-pr` resource in
turn must run a `check` for every space.

When any of the space combinations of `atc-pr-unit` has a new version
available, I expect a build of `atc-pr-unit` to automatically kick off.

1. The `github-pr` space type is implemented like so:
  * `check` returns the set of PRs, e.g. `[{"remote": "pull/23/head"}]`
  * `put` is used to update the PR status
  * `get` returns metadata pertaining to the PR
  * `delete` is unimplemented (maybe it closes the pr? not important here.)

1. The `atc-prs` space is configured to detect the spaces of the
  `concourse/atc` repo.

1. The `atc-pr` resource is configured as a regular old `git` resource,
  performing `checks` across all spaces returned by `atc-prs`.

  * This works by having one *resource* `check` for each space returned by
    `atc-prs`, by merging the space object `{"remote":"pull/23/head"}` with the
    rest of the resource config.

    The `git` resource would have to understand this config param to fetch the
    given remote and emit the version.

  * Because this resource is configured `across: atc-prs`, any use of the
    resource must be space-aware.

1. The `atc-pr-unit` job configures a `get` step of `atc-pr`. Because the
  `atc-pr` resource is spatial, each space satisfying the `spaces: all` filter
  results in a "space combination" for the job, each with independent
  scheduling.


### Runtime
