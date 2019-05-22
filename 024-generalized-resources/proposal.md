# Generalized Resources

This proposal adapts today's 'resource' interface ("v1") into a more general resource interface that is less specialized to the 'versioned artifacts' use case currently supported by Concourse pipelines.

The proposed interface is versioned, starting at `2.0`. Each resource type will have its [resource info](#resource-info) fetched to discover its version.

Concourse will support use of mixed resource interface versions. When a resource type does not support the new 'info' request flow it will be assumed to be v1.

[Resource actions](#resource-actions) have all been updated to fit the exact same request/response interface. This proposal defines four core actions: `check`, `get`, `put`, and `delete`.

Each action emits **config fragments**, which replace 'versions' from the v1 interface. By using a more general name the interface can be used for use cases beyond just versioning artifacts. See [Interpreting the Resource Interface](#interpreting-the-resource-interface) for more details.

## Previous Discussions

* [RFC #1](https://github.com/concourse/rfcs/pull/1), now defunct, is similar to this proposal but had a concept of "spaces" baked into the interface. This concept has been decoupled from the resource interface and will be re-introduced in a later RFC that is compatible with v1 and v2 resources.
  * **Recommended reading**: [this comment](https://github.com/concourse/rfcs/pull/1#issuecomment-477749314) outlines the thought process that led to this RFC.
* [concourse/concourse#534](https://github.com/concourse/concourse/issues/534) was the first 'new resource interface' proposal which pre-dated the RFC process.

## Motivation

* Support for creating multiple versions from `put`: [concourse/concourse#2660](https://github.com/concourse/concourse/issues/2660)

* Support for deleting versions: [concourse/concourse#362](https://github.com/concourse/concourse/issues/362), [concourse/concourse#524](https://github.com/concourse/concourse/issues/524)

* Having resource metadata immediately available via check: [concourse/git-resource#193](https://github.com/concourse/git-resource/issues/193), [concourse/concourse#1714](https://github.com/concourse/concourse/issues/1714)

* Unifying `source` and `params` as just `config` so that resources don't have to care where configuration is being set in pipelines: [concourse/git-resource#172](https://github.com/concourse/git-resource/pull/172), [concourse/bosh-deployment-resource#13](https://github.com/concourse/bosh-deployment-resource/issues/13), [concourse/bosh-deployment-resource#6](https://github.com/concourse/bosh-deployment-resource/issues/6), [concourse/cf-resource#20](https://github.com/concourse/cf-resource/pull/20), [concourse/cf-resource#25](https://github.com/concourse/cf-resource/pull/25), [concourse/git-resource#210](https://github.com/concourse/git-resource/pull/210)

* Make resource actions reentrant so that we no longer receive `unexpected EOF` errors when reattaching to an in-flight build whose resource action completed while we weren't attached: [concourse/concourse#1580](https://github.com/concourse/concourse/issues/1580)

* Support for showing icons for resources in the web UI: [concourse/concourse#788](https://github.com/concourse/concourse/issues/788), [concourse/concourse#3220](https://github.com/concourse/concourse/pull/3220), [concourse/concourse#3581](https://github.com/concourse/concourse/pull/3581)

* Standardize TLS configuration so every resource doesn't implement their own way: [concourse/rfcs#9](https://github.com/concourse/rfcs/issues/9)

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
  * `info`: given a **config**, emit a response specifying the command to run for each action
  * `check`: given a **config**, emit **config fragments**
  * `get`: given a **config**, populate a directory with **bits**
  * `put`: given a **config** and a directory containing **bits**, create or update **config fragments**
  * `delete`: given a **config** and a directory containing **bits**, delete **config fragments**
  * Examples:
    * `git-branches` resource type for tracking branches in a repo
    * `git` resource type for tracking commits in a branch
    * `github-status` resource type for emitting build status notifications for commits
    * `time` resource type for doing timed job triggers

* **Resource**: a **resource type** with a user-provided **config**, used together to represent external state.

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

// TLSConfig captures common configuration for communicating with servers over TLS.
type TLSConfig struct {
  // An array of CA certificates to trust.
  CAs []string `json:"ca_certs,omitempty"`

  // Skip certificate verification, effectively making communication insecure.
  SkipVerification bool `json:"skip_verification,omitempty"`
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
  //
  // Icons must be namespaced by in order to explicitly reference an icon set
  // supported by Concourse, e.g. 'mdi:' for Material Design Icons.
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

  // Configuration for handling TLS.
  TLS TLSConfig `json:"tls,omitempty"`

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

## Resource Info

Prior to running any action, Concourse will execute the default command (i.e. [`CMD`](https://docs.docker.com/engine/reference/builder/#cmd)) for the image with an `InfoRequest` piped to `stdin`.

The command must write an `InfoResponse` to `stdout` in response. This response specifies the resource interface version that the resource type conforms to, an optional icon to show in the UI, and the command to run for each supported resource action.

### Example **info** request/response

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
  "icon": "mdi:github-circle",
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

## Interpreting the Resource Interface

The resource interface itself is now a general way of expressing interactions with external state. It is no longer based on versioning of artifacts.

Concourse will now codify things like "artifact resources" and "spatial resources" as interpretations of this general interface, composing resource types with one another via **config fragments**.

By leveraging composition instead of having a monolithic interface, this approach encourages narrowly scoped resource type implementations. Implementations that are small in scope are more likely to be correct and 'finished' at some point, and only become more powerful as Concourse enables new workflows at the pipeline-level.

For example, as an author of a `git` artifact resource type, I just have to implement a linear, versioned artifact interface. In `git`, linear versioning (assuming `--first-parent`) occurs on a branch, so that can be specified in `config`. What if users want to run against all branches? As a resource type author, that's not my problem! Let them use a `git-branches` spatial resource and compose it with my resource to provide the `branch` config.

By not baking the workflow directly into the interface, this also allows a single resource type implementation to be used for multiple interpretations. For example, notifications and triggers are complementary and can both be supported by a resource type that supports the required actions (`put` for notifications and `check`/`get` for triggers).

Four interpretations are outlined in the following proposals. The RFCs linked below show how the v2 interface can be used to support various pipeline workflows. The described workflows can also work with v1 resources, however, and so they are not explicitly dependent on the v2 interface. This is done intentionally so that our roadmap doesn't have to be so linear.

### [Artifact resources](https://github.com/concourse/rfcs/pull/26)

```yaml
artifacts:
- name: concourse
  type: git
  source:
    uri: https://github.com/concourse/concourse
    branch: master
```

* `check`: return versions in order
* `get`: fetch a version of the resource
* `put`: push versions of a resource
* `delete`: delete versions of a resource

Examples: `git`

### [Spatial resources](https://github.com/concourse/rfcs/pull/29)

```yaml
spaces:
- name: concourse-branches
  type: git-branches
  source:
    uri: https://github.com/concourse/concourse
```

* `check`: return a fragment for each space, no order
* `get`: fetch whatever metadata is useful for a given space
* `put`: create or update spaces
* `delete`: delete spaces

Examples: `git-branches`, `github-prs`

### [Notification resources](https://github.com/concourse/rfcs/pull/28)

```yaml
notifications:
- name: concourse-status
  type: github-status
  source:
    repository: concourse/concourse
    access_token: abcdef
```

* `check`: not used
* `get`: fetch bits pertaining to the notification
* `put`: emit a notification
* `delete`: not sure - clear github status?

Examples: `github-status`, `slack`

### [Trigger resources](https://github.com/concourse/rfcs/pull/27)

```yaml
triggers:
- name: every-10m
  type: time
  source: {interval: 10m}
```

* `check`: check against last fragment used for job
* `get`: fetch bits pertaining to the trigger
* `put`: not useful
* `delete`: not useful

Examples: `time`

## Out of scope

* [Richer metadata](https://github.com/concourse/concourse/issues/310) - this hasn't gained much traction and probably needs more investigation before it can be incorporated. This should be easy enough to add as a later RFC.
