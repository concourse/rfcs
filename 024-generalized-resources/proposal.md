# Generalized Resources

This proposal adapts today's 'resource' interface into a more general interface that is less specialized to the 'versioned artifacts' use case, while also introducing versioning to the interface so that it'll be easier to make incremental changes to the interface itself and support mixed resource versions within a user's pipeline.

Today's resources are closely tied to the 'versioned artifact' use case, so this proposal will also show how the new interface is "interpreted" in order to support this use case.

## Motivation

* Support for deleting versions in `put`:
  * https://github.com/concourse/concourse/issues/362
  * https://github.com/concourse/concourse/issues/524

* Support for creating multiple versions from `put`:
  * https://github.com/concourse/concourse/issues/2660

* Having resource metadata immediately available via check:
  * https://github.com/concourse/git-resource/issues/193
  * https://github.com/concourse/concourse/issues/1714

* Make the `get` after `put` opt-in:
  * https://github.com/concourse/concourse/issues/3299
  * https://stackoverflow.com/questions/38964299/why-does-concourse-get-a-resource-after-puting-it
  * https://github.com/concourse/registry-image-resource/issues/16

* Unifying `source` and `params` as just `config` so that resources don't have to care where configuration is being set in pipelines:
  * https://github.com/concourse/git-resource/pull/172
  * https://github.com/concourse/bosh-deployment-resource/issues/13
  * https://github.com/concourse/bosh-deployment-resource/issues/6
  * https://github.com/concourse/cf-resource/pull/20
  * https://github.com/concourse/cf-resource/pull/25
  * https://github.com/concourse/git-resource/pull/210

* Generalize interface to support non-versioned state:
  * https://github.com/concourse/concourse/issues/739

* Support for trigger-only resources that don't result in fetching anything - they just trigger the job:
  * https://github.com/concourse/concourse/issues/3572
  * https://github.com/concourse/concourse/issues/3595

* Make resource actions more reentrant so that we no longer receive `unexpected EOF` errors upon reattaching to an in-flight build.
  * https://github.com/concourse/concourse/issues/1580

* Support multi-branch workflows:
  * https://github.com/concourse/concourse/issues/1172

* Begin phasing out `version: every` in by reframing the problem as 'pipeline per commit':
  * https://github.com/concourse/concourse/issues/736

* Support notifications in a way that doesn't pollute pipeline config and UI:
  * https://github.com/concourse/concourse/issues/1052
  * https://github.com/concourse/rfcs/issues/10

* Support for showing icons for resources in the web UI:
  * https://github.com/concourse/concourse/issues/788
  * https://github.com/concourse/concourse/pull/3220
  * https://github.com/concourse/concourse/pull/3581

## Glossary

* **Config**: an arbitrarily nested JSON object containing user-provided configuration
  * Examples: `{"uri":"https://github.com/concourse/concourse"}`, `{"interval":"10m"}`

* **Config fragment**: a smaller JSON object intended to be "spliced" into a **config** by assigning each field from the fragment into the config.
  * Examples: `{"ref":"abcdef"}`, `{"branch":"develop"}`

* **Bits**: a directory containing arbitrary data
  * Examples: source code, compiled artifacts, etc.

* **Metadata**: structured data associated to a **config fragment** containing information about the fragment that should be surfaced to the user
  * Examples: `[{"name":"committer","value":"Alex Suraci"}]`

* **Resource type**: an implementation of the interface defined by this proposal, typically provided as a container image. Implements the following actions:
  * `info`: given a **config**, emit provide the commands to run for the following actions:
  * `check`: given a **config**, emit **config fragments**
  * `get`: given a **config**, populate a directory with **bits**
  * `put`: given a **config** and a directory containing **bits**, create or delete **config fragments**
  * Examples:
    * `git-branches` resource type for tracking branches in a repo
    * `git` resource type for tracking commits in a branch
    * `github-status` resource type for emitting build status notifications for commits
    * `time` resource type for doing timed job triggers

* **Resource**: a **resource type** with a user-provided **config**, used together to represent external state.

## Example Resources

```yaml
type: git-branches
source:
  uri: https://github.com/concourse/concourse
```

```yaml
type: git
source:
  uri: https://github.com/concourse/concourse
  branch: master
```

```yaml
type: github-status
source:
  repository: concourse/concourse
  access_token: abcdef
```

```yaml
type: time
source:
  interval: 10m
```

## Open Questions

* enrich metadata?
  * https://github.com/concourse/concourse/issues/310
* standardize TLS config?
  * https://github.com/concourse/rfcs/issues/9
* webhooks?
  * should these instead be something supported by *Concourse*?
  * can this come from `/info`? would it need `config` passed to it?
* resource-determined triggerability of versions?
  * https://github.com/concourse/rfcs/issues/11

## Answered Questions

* Version filtering is probably best left to `config`.

## New Implications

many

## Yet-to-be-organized notes

Cataloguing ways in which generalized resources can be composed to accomplish different goals:

* artifact `check` -> config fragment + artifact `check`
  * check from version
* artifact `check` -> config fragment + artifact `get`
  * fetch specific version
* artifact `put` -> config fragment + artifact `get`
  * fetch just-created version
* spatial `check` -> config fragment + artifact `check`
  * check across all spaces
* artifact `get` -> config fragment + notification `put`
  * update github status
* trigger `check` -> config fragment -> trigger build if different from last config fragment
  * trigger-only resources
  * maybe the config fragment could be passed to something to support parameterized triggers? :thinking:
  * maybe that could fit nicely with however we approach https://github.com/concourse/concourse/issues/783? :thinking:
