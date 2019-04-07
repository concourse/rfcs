# Generalized Resources

This proposal adapts today's 'resource' interface into a more general interface that is less specialized to the 'versioned artifacts' use case, while also introducing versioning to the interface so that it'll be easier to make incremental changes to the interface itself and support mixed resource versions within a user's pipeline.

Today's resources are closely tied to the 'versioned artifact' use case, so this proposal will also show how the new interface is "interpreted" in order to support this use case.

## Motivation

* Support for deleting versions in `put`:
  * [concourse/concourse#362](https://github.com/concourse/concourse/issues/362)
  * [concourse/concourse#524](https://github.com/concourse/concourse/issues/524)

* Support for creating multiple versions from `put`:
  * [concourse/concourse#2660](https://github.com/concourse/concourse/issues/2660)

* Having resource metadata immediately available via check:
  * [git-resource#193](https://github.com/concourse/git-resource/issues/193)
  * [concourse/concourse#1714](https://github.com/concourse/concourse/issues/1714)

* Make the `get` after `put` opt-in:
  * [concourse/concourse#3299](https://github.com/concourse/concourse/issues/3299)
  * [image-resource#16](https://github.com/concourse/registry-image-resource/issues/16)

* Unifying `source` and `params` as just `config` so that resources don't have to care where configuration is being set in pipelines:
  * [concourse/git-resource#172](https://github.com/concourse/git-resource/pull/172)
  * [concourse/bosh-deployment-resource#13](https://github.com/concourse/bosh-deployment-resource/issues/13)
  * [concourse/bosh-deployment-resource#6](https://github.com/concourse/bosh-deployment-resource/issues/6)
  * [concourse/cf-resource#20](https://github.com/concourse/cf-resource/pull/20)
  * [concourse/cf-resource#25](https://github.com/concourse/cf-resource/pull/25)
  * [concourse/git-resource#210](https://github.com/concourse/git-resource/pull/210)

* Generalize interface to support non-versioned state:
  * [concourse/concourse#739](https://github.com/concourse/concourse/issues/739)

* Support for trigger-only resources that don't result in fetching anything - they just trigger the job:
  * [concourse/concourse#3572](https://github.com/concourse/concourse/issues/3572)
  * [concourse/concourse#3595](https://github.com/concourse/concourse/issues/3595)

* Make resource actions more reentrant so that we no longer receive `unexpected EOF` errors upon reattaching to an in-flight build.
  * [concourse/concourse#1580](https://github.com/concourse/concourse/issues/1580)

* Support multi-branch workflows:
  * [concourse/concourse#1172](https://github.com/concourse/concourse/issues/1172)

* Begin phasing out `version: every` in by reframing the problem as 'pipeline per commit':
  * [concourse/concourse#736](https://github.com/concourse/concourse/issues/736)

* Support notifications in a way that doesn't pollute pipeline config and UI:
  * [concourse/concourse#1052](https://github.com/concourse/concourse/issues/1052)
  * [concourse/rfcs#10](https://github.com/concourse/rfcs/issues/10)

* Support for showing icons for resources in the web UI:
  * [concourse/concourse#788](https://github.com/concourse/concourse/issues/788)
  * [concourse/concourse#3220](https://github.com/concourse/concourse/pull/3220)
  * [concourse/concourse#3581](https://github.com/concourse/concourse/pull/3581)

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

## Interface Types

```go
// Config represents arbitrary user-specified configuration.
type Config map[string]interface{}

// ConfigFragment represents additional fields that can be spliced into a Config.
type ConfigFragment map[string]interface{}

// MetadataField represents a named bit of metadata associated to a ConfigFragment.
type MetadataField struct {
  Name  string `json:"name"`
  Value string `json:"value"`
}

// InfoRequest is the payload written to stdin for the `./info` script.
type InfoRequest struct {
  // User-specified configuration.
  Config Config `json:"config"`
}

// InfoResponse is the payload written to stdout from the `./info` script.
type InfoResponse struct {
  // The version of the resource interface that this resource type conforms to.
  InterfaceVersion string `json:"interface_version"`

  // An optional icon name to show to the user when viewing the resource.
  Icon string `json:"icon,omitempty"`

  // The actions supported by the resource type.
  Actions struct {
    // Command to run when performing check actions.
    Check string `json:"check,omitempty"`

    // Command to run when performing get actions.
    Get string `json:"get,omitempty"`

    // Command to run when performing put actions.
    Put string `json:"put,omitempty"`

    // Command to run when performing delete actions.
    Delete string `json:"delete,omitempty"`
  } `json:"actions"`
}

// ActionRequest is the payload written to stdin for each action command.
type ActionRequest struct {
  // User-specified configuration.
  Config Config `json:"config"`

  // Path to a file into which the action must write its response.
  ResponsePath string `json:"response_path"`
}

// ActionResponse is written to the `response_path` by an action for each fragment affected by the action. Multiple respones may be written as a JSON stream.
type ActionResponse struct {
  // The fragment. May be used as an identifier, unique within the scope of a Config.
  Fragment ConfigFragment `json:"fragment"`

  // Metadata to associate with the fragment. Shown to the user.
  Metadata []MetadataField `json:"metadata,omitempty"`
}
```

## `info`: discover resource type implementation info

Prior to running any action, `./info` will be executed with an `InfoRequest` piped to `stdin`. The path to this script will be relative to the image's working directory.

The resource type must emit a `InfoResponse` to `stdout` in response. This response specifies the resource interface version that the resource conforms to, an optional icon to show in the UI, and the command to run for each supported resource action.

### Example `info` request-response

Request sent to `stdin`:

```json
{
  "config": {
    "uri": "https://github.com/concourse/concourse"
  }
}
```

Response written to `stdout`:

```json
{
  "interface_version": "2.0",
  "icon": "github-circle",
  "actions": {
    "check": "/usr/bin/git-resource check",
    "get": "/usr/bin/git-resource get",
    "put": "/usr/bin/git-resource put",
    "delete": "/usr/bin/git-resource delete"
  }
}
```

## Resource Actions

Each action is invoked with a JSON-encoded `ActionRequest` piped to `stdin`. This request contains the **config** and the path to which the response should be written. This path may be relative, and if so it is to be expanded from the current working directory.

All actions will be run in a working directory for the **bits** - either an empty directory to which bits should be written, or a directory containing the bits given to the action.

All actions respond by performing their side-effect and writing sequential `ActionResponse` JSON objects to the file path specified by `response_path`. How this response is interpreted depends on the action, but typically there should be one response for each external resource affected (`put`, `delete`), discovered (`check`), or fetched (`get`).

### Example **action** request/response

Request sent to `stdin`:

```json
{
  "config": {
    "uri": "https://github.com/concourse/rfcs",
    "branch": "master"
  },
  "response_path": "../response/response.json"
}
```

Response written to `../response/response.json`:

```json
{
  "fragment": {"ref": "e4be0b367d7bd34580f4842dd09e7b59b6097b25"},
  "metadata": [
    {
      "name": "message",
      "value": "init"
    }
  ]
}
{
  "fragment": {"ref": "5a052ba6438d754f73252283c6b6429f2a74dbff"},
  "metadata": [
    {
      "name": "message",
      "value": "add not-very-useful-yet readme"
    }
  ]
}
{
  "fragment": {"ref": "2e256c3cb4b077f6fa3c465dd082fa74df8fab0a"},
  "metadata": [
    {
      "name": "message",
      "value": "start fleshing out RFC process"
    }
  ]
}
```

This response would be typical of a `check` that ran against a repo that had three commits.

### `check`: monitor a config to discover config fragments

* **bits** is initially empty - the resource may place arbitrary data there to cache state between checks

### `get`: fetch bits corresponding to a spliced config

* **bits** is initially empty, and the action must fetch into it

### `put`: create-or-update config fragments

* **bits** contains arbitrary user-provided data

### `delete`: use bits to delete config fragments

* **bits** contains arbitrary user-provided data

## Artifact resources with v2

All v1 resources are effectively "versioned artifact resources", as that is the only way Concourse pipelines support using them.

A v2 resource type can be used as a versioned artifact resource by treating the **config fragments** as **versions** and implementing the following behavior:

### `check`: discover versions over time

The `check` action will first be run with a "naked" config, containing only what the user specified. In this situation `check` must return *all* versions discovered in the config, in chronological order.

Subsequent calls to `check` will be given a config that has been spliced with the last emitted version. The `check` script must emit any versions that came after the specified version.

If the specified version is no longer present, the `check` action must emit a `reset` event and then return *all* versions, as if the version was not specified in the first place.

### `get`: fetch a version of an artifact

The `get` action will always be invoked with a spliced config specifying which version to fetch. It is given an empty directory in which it should fetch the bits.

A `fetched` event must be emitted for all versions that have been fetched into the bits directory. Each version will be recorded as an input to the build.

### `put`: create, update, and/or delete artifact versions

The `put` action will be invoked with user-provided configuration and arbitrary bits.

A `created` event must be emitted for all versions that have been created by the `put` action. These will be recorded as outputs of the build.

A `deleted` event must be emitted for all versions that have been deleted by the `put` action. These versions will be marked "deleted" and no longer be available for use in other builds.

## Open Questions

* [enrich metadata?](https://github.com/concourse/concourse/issues/310)
* [standardize TLS config?](https://github.com/concourse/rfcs/issues/9)
* [resource-determined triggerability of versions?](https://github.com/concourse/rfcs/issues/11)
* webhooks?
  * should these instead be something supported by *Concourse*?

## Answered Questions

* Version filtering is probably best left to `config`.

## New Implications

* Notifications
* Spaces
* Triggers

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
  * maybe that could fit nicely with however we approach [concourse/concourse#738](https://github.com/concourse/concourse/issues/783)? :thinking:
