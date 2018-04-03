# Summary

1. Tighten up the existing resource interface to better represent its intended
   use case: versioned objects, changing over time, with an external source of
   truth.

1. Introduce versioning to the resource interface, so that we can maintain
   backwards-compatibility.

1. Introduce new interfaces to support today's use cases that are currently
   being shoehorned into the resource interface.

1. Provide an answer for how spaces are discovered and used, in light of
   concourse/concourse#1707.


# Motivation

Resources today are used for many things that they should not be used for. This
leads to surprising behavior, difficult workarounds, and a lack of consistency
in what you can expect from any given resource type.

Many of them only implement a subset of the interface, or only provide stubs
for part of it. This makes them not a true resource.

We have a need in concourse/concourse#1707 to discover change over *space*, not
*time*, which requires additions or changes to the resource interface. We also
have



* External un-versioned resources that have a "current state" (i.e. a deployment, a set of git branches, a set of git PRs)
* Push-only things with no state, like notifications
* Un-versioned blobs that can be passed around and deleted

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

check  : Config -> List Version
get    : Config -> Version -> Bits
put    : Config -> Bits -> Set Version
delete : Config -> Bits -> Set Version

-- Spaces (BOSH deployments, Git branches, Git PRs, Semver trees)
type alias Space = Dict String String

check  : Config -> Set Space
get    : Config -> Space -> Bits
put    : Config -> Bits -> Set Space
delete : Config -> Bits -> Set Space

-- Notifications (slack, email)
type alias BuildMetadata =
  { buildStatus : String
  , buildNumber : Int
  , jobName : String
  , pipelineName : String
  , teamName : String
  }

notify : Config -> BuildMetadata -> ()
```


# Examples

- Pull Requests
- BOSH deploys
- Feature branches
- Arbitrary branches
- IaaSes


## Pull Requests

```yaml
space_types:
- name: github-pr
  type: docker-image
  source: {repository: concourse/github-pr-space-type}

spaces:
- name: atc-prs
  type: github-pr
  source: {repository: concourse/atc}

resources:
- name: atc-pr
  type: git
  source: {uri: "https://github.com/concourse/atc"}
  spaces: atc-prs

jobs:
- name: atc-pr-unit
  plan:
  - get: atc-pr
    trigger: true
    spaces: all
  - task: unit
    file: atc/ci/pr.yml
```

!! stateful spaces? hooks on build start/finish with state available
!! this could be used to reflect PR status, send slack alerts, track pool entries



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
