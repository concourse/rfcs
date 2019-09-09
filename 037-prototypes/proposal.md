# Concourse Prototype Protocol

This proposal outlines a small protocol for invoking conceptually related
commands in a container image with a JSON-based request-response. This protocol
is to be used as a foundation for both a new resource type interface and as a
way to package and reuse arbitrary functionality (i.e. tasks).

The protocol is versioned, starting at `1.0`.

An implementation of this protocol is called a **prototype**. ([Why
'prototype'?](#etymology)) Conceptually, a prototype handles **messages** sent
to **objects** which are provided by the user as configuration or produced by
the prototype itself in response to a prior message.

For example, a `git` prototype may handle a `check` message against a
repository object to produce commit objects, which then can then handle a `get`
message to fetch the commit.

An object's supported messages are discovered through an [Info
request](#prototype-info). For example, the `git` prototype may support `get`
against a commit object, but not against a repository object. The Info request
is also a good place to perform validation.

A message is handled by invoking its corresponding command in the container
with a [Message request](#prototype-message). A message handler responds by
emitting objects and accompanying metadata. These objects are written to a
response path specified by the request.

While a prototype can handle any number of messages with any name, certain
messages names will have semantic meaning in a Concourse pipeline. For example,
a prototype which supports `check` and `get` messages can be used as a resource
in a pipeline. These messages will be sent by the Concourse pipeline scheduler
if a user configures the prototype and config as a resource.


## Previous Discussions

* [RFC #24][rfc-24] was a previous iteration of this proposal. It had a fixed
  set of "actions" (`check`, `get`, `put`, `delete`) and was still centered in
  the 'resources' concept and terminology. This RFC generalizes the design
  further, switches to prototype-based terminology and allows prototypes to
  handle arbitrary messages.

* [RFC #1][rfc-1], now defunct, is similar to this proposal but had a concept
  of "spaces" baked into the interface. This concept has been decoupled from
  the resource interface and will be re-introduced in a later RFC that is
  compatible with v1 and v2 resources.
  * **Recommended reading**: [this comment][rfc-1-comment] outlines the thought
    process that led to this RFC.

* [concourse/concourse#534](https://github.com/concourse/concourse/issues/534)
  was the first 'new resource interface' proposal which pre-dated the RFC
  process.


## Motivation

* Support for 'reusable tasks' in order to stop the shoehorning of
  functionality into the resource interface:
  [concourse/rfcs#7](https://github.com/concourse/rfcs/issues/7).

* Support for creating multiple versions from `put`:
  [concourse/concourse#2660](https://github.com/concourse/concourse/issues/2660)

* Support for deleting versions:
  [concourse/concourse#362](https://github.com/concourse/concourse/issues/362),
  [concourse/concourse#524](https://github.com/concourse/concourse/issues/524)

* Having resource metadata immediately available via check:
  [concourse/git-resource#193](https://github.com/concourse/git-resource/issues/193),
  [concourse/concourse#1714](https://github.com/concourse/concourse/issues/1714)

* Unifying `source` and `params` as just `config` so that resources don't have
  to care where configuration is being set in pipelines:
  [concourse/git-resource#172](https://github.com/concourse/git-resource/pull/172),
  [concourse/bosh-deployment-resource#13](https://github.com/concourse/bosh-deployment-resource/issues/13),
  [concourse/bosh-deployment-resource#6](https://github.com/concourse/bosh-deployment-resource/issues/6),
  [concourse/cf-resource#20](https://github.com/concourse/cf-resource/pull/20),
  [concourse/cf-resource#25](https://github.com/concourse/cf-resource/pull/25),
  [concourse/git-resource#210](https://github.com/concourse/git-resource/pull/210)

* Make resource actions reentrant so that we no longer receive `unexpected EOF`
  errors when reattaching to an in-flight build whose resource action completed
  while we weren't attached:
  [concourse/concourse#1580](https://github.com/concourse/concourse/issues/1580)

* Support for showing icons for resources in the web UI:
  [concourse/concourse#788](https://github.com/concourse/concourse/issues/788),
  [concourse/concourse#3220](https://github.com/concourse/concourse/pull/3220),
  [concourse/concourse#3581](https://github.com/concourse/concourse/pull/3581)

* Standardize TLS configuration so every resource doesn't implement their own
  way: [concourse/rfcs#9](https://github.com/concourse/rfcs/issues/9)


## Glossary

* **Prototype**: an implementation of the interface defined by this proposal,
  typically packaged as an OCI container image.

  Examples:

  * a `git` prototype for interacting with commits and branches in a repo.
  * a `time` prototype for performing interval triggers.

* **Message**: an operation to perform against the object. Corresponds to a
  command to run in the container image.

  Examples:

  * `check`
  * `get`

* **Object**: a logical entity, encoded as a JSON object, acted on and emitted
  by a prototype in response to messages.

  Examples:

  * `{"uri":"https://github.com/concourse/rfcs"}`
  * `{"ref":"e4be0b367d7bd34580f4842dd09e7b59b6097b25"}`

* **Metadata**: additional information about an object to surface to users.

  Examples:

  * `[{"name":"committer","value":"Alex Suraci"}]`

* **Bits**: a directory containing arbitrary data.

  Examples:
  
  * source code checked out from a repo
  * compiled artifacts created by a task or another prototype

* **Resource**: a **prototype** which implements the following messages:

  * `check`: discover objects representing versions, in order
  * `get`: fetch an object version


## Interface Types

```go
// Object is an object receiving messages and being emitted in response to
// messages.
type Object map[string]interface{}

// InfoRequest is the payload written to stdin for the `./info` script.
type InfoRequest struct {
  // The properties specifying the object to act on.
  Object Object `json:"object"`

  // Configuration for handling TLS.
  TLS TLSConfig `json:"tls,omitempty"`
}

// InfoResponse is the payload written to stdout from the `./info` script.
type InfoResponse struct {
  // The version of the prototype interface that this prototype conforms to.
  InterfaceVersion string `json:"interface_version"`

  // An optional icon to show to the user.
  //
  // Icons must be namespaced by in order to explicitly reference an icon set
  // supported by Concourse, e.g. 'mdi:' for Material Design Icons.
  Icon string `json:"icon,omitempty"`

  // The messages supported by the object.
  Messages []string `json:"messages,omitempty"`
}

// MessageRequest is the payload written to stdin for a message.
type MessageRequest struct {
  // The properties specifying the object to act on.
  Object Object `json:"object"`

  // Configuration for handling TLS.
  TLS TLSConfig `json:"tls,omitempty"`

  // Path to a file into which the message handler must write its response.
  ResponsePath string `json:"response_path"`
}

// MessageResponse is written to the `response_path` for each object returned
// by the message. Multiple responses may be written to the same file,
// concatenated as a JSON stream.
type MessageResponse struct {
  // The identifying properties for the object.
  Object Object `json:"object"`

  // Metadata to associate with the properties. Shown to the user.
  Metadata []MetadataField `json:"metadata,omitempty"`
}

// TLSConfig captures common configuration for communicating with servers over
// TLS.
type TLSConfig struct {
  // An array of CA certificates to trust.
  CACerts []string `json:"ca_certs,omitempty"`

  // Skip certificate verification, effectively making communication insecure.
  SkipVerification bool `json:"skip_verification,omitempty"`
}

// MetadataField represents a named bit of metadata associated to an object.
type MetadataField struct {
  Name  string `json:"name"`
  Value string `json:"value"`
}
```


## Prototype Info

Prior to sending any message, the default command (i.e.
[`CMD`](https://docs.docker.com/engine/reference/builder/#cmd)) for the image
will be executed with an `InfoRequest` piped to `stdin`. This request contains
an **object**.

The command must write an `InfoResponse` to `stdout` in response. This response
specifies the prototype interface version that the prototype conforms to, an
optional icon to show in the UI, and the messages supported by the given
object.

### Example **info** request/response

Request sent to `stdin`:

```json
{ "object": { "uri": "https://github.com/concourse/concourse" } }
```

Response written to `stdout`:

```json
{
  "interface_version": "1.0",
  "icon": "mdi:github-circle",
  "messages": ["check"]
}
```


## Prototype Messages

A message handler is invoked by executing the command named after the message
with a JSON-encoded `MessageRequest` piped to `stdin`. This request contains
the **object** which is conceptually receiving the message.

All message commands will be run in a working directory layout containing the
**bits** provided to the message handler, as well as empty directories to which
bits should be written by the message handler.

A message handler writes its `MessageResponse`(s) to the file path specified by
`response_path` in the `MessageRequest`. This path may be relative to the
initial working directory.

How this response is interpreted depends on the message, but typically there
should be one response for each object affected (`put`, `delete`), discovered
(`check`), or fetched (`get`).

### Example **message** request/response

Request sent to `stdin`:

```json
{
  "object": {
    "uri": "https://github.com/concourse/rfcs",
    "branch": "master"
  },
  "response_path": "../response/response.json"
}
```

Response written to `../response/response.json`:

```json
{
  "object": {"ref": "e4be0b367d7bd34580f4842dd09e7b59b6097b25"},
  "metadata": [ { "name": "message", "value": "init" } ]
}
{
  "object": {"ref": "5a052ba6438d754f73252283c6b6429f2a74dbff"},
  "metadata": [ { "name": "message", "value": "add not-very-useful-yet readme" } ]
}
{
  "object": {"ref": "2e256c3cb4b077f6fa3c465dd082fa74df8fab0a"},
  "metadata": [ { "name": "message", "value": "start fleshing out RFC process" } ]
}
```

This response would be typical of a `check` that ran against a `git` repository
that had three commits.


## Object Cloning

Technically, all parts of the prototocol have been specified, but there is a
particular usage pattern that is worth outlining here as it relates to the
'prototype' terminology and may be common across a few use cases for this
protocol.

**Objects** are just arbitrary JSON data. This data is consumed by a prototype
action and may be emitted by the prototype in response to a message.

One example of a message that will yield new objects is `check` (part of the
[resource interface](https://github.com/concourse/rfcs/pull/26)). This message
will be sent to prototypes that are used as a resource in a Concourse pipeline.

Per the resource interface, the `check` message returns an object for each
version. However, these objects aren't self-contained - instead, they are an
implicit *clone* of the original object, with the returned object's fields
merged in.

For example, the `git` prototype may be initially receive the following
user-configured properties in a `check` message:

```json
{
  "object": {
    "uri": "https://github.com/concourse/rfcs",
    "branch": "master"
  }
}
```

Let's say the `check` handler responds with the following objects, representing
two commits in the repository (with `metadata` omitted for brevity):

```json
{
  "object": {
    "ref": "e4be0b367d7bd34580f4842dd09e7b59b6097b25"
  }
}
{
  "object": {
    "ref": "5a052ba6438d754f73252283c6b6429f2a74dbff"
  }
}
```

Per the resource interface, these object properties would be saved as versions
of the resource in the order returned by `check`.

When it comes time to fetch a version in a build, the `git` prototype would
receive a `get` message with the following object:

```json
{
  "object": {
    "uri": "https://github.com/concourse/rfcs",
    "branch": "master",
    "ref": "e4be0b367d7bd34580f4842dd09e7b59b6097b25"
  }
}
```

Note that the object properties have been merged into the original set of
properties.

## Migrating Resources to Prototypes

All existing Concourse resources can map directly to a prototype
implementation, as they implement a set of commands which act on the resource
type's defining entity.

Concourse will support use of resources alongside prototypes. When a resource
type does not support the new 'info' request flow it will be assumed to be v1.

* `resource_types`: v1 resources
* `prototypes:` prototypes!

## Direct Pipeline Usage

Now for the fun part!

So far this proposal has been waxing theoretical and using a bunch of
fancy-pants terminology. But how will it look when it goes up to the user?

For starters, I *don't* think we should necessarily bubble up the same
terminology. Users writing pipelines may appreciate the mental model underlying
everything, but it's more likely that when they're writing build plans they
just want to run something. Not "send something a message."

There are a lot of options here. Some of them add a bit of syntactic sugar
(e.g. `oci-image.build`) which is uncharted territory but does feel a bit
nicer.

```yaml
prototypes:
- name: oci-image
  type: registry-image
  source:
    repository: vito/oci-image-prototype

jobs:
- name: build-image
  plan:
  - get: concourse

  # manually run 'get' action anonymously
  #
  # NOTE: has no checking or caching semantics
  - run: get
    type: git
    params:
      uri: https://github.com/concourse/concourse
      ref: v5.0.0

    # 'get' handlers always produce an output named 'resource'
    output_mapping: {resource: concourse}
      
  - run: build
    type: oci-image
    params: {context: concourse}
    outputs: [image]
```

### Running a Message Handler

A new `run` step type will be introduced. It takes the name of a message to
send, and a `type` denoting the prototype to use:

```yaml
prototypes:
- name: some-prototype
  type: registry-image
  source:
    repository: vito/some-prototype

jobs:
- name: run-prototype
  plan:
  - run: some-message
    type: some-prototype
```

So far this example is not very useful. There aren't many things I can think to
run that don't take any inputs or configuration or produce any outputs. Let's
try giving it an input and collecting its resulting output. This can be done
with `inputs`/`input_mapping` and `outputs`/`output_mapping`:

```yaml
prototypes:
- name: oci-image
  type: registry-image
  source:
    repository: vito/oci-image-prototype

resources:
- name: my-image-src
  type: git
  source:
    uri: https://example.com/my-image-src

jobs:
- name: build-my-image
  plan:
  - get: my-image-src
  - run: build
    type: oci-image
    input_mapping: {context: my-image-src}
    outputs: [image]
```

For this example we've also snuck in a pipeline resource definition
(`my-image-src`). Under the hood, resources are just prototypes which support
`check` and `get` and have the following behavior:

* A resource prototype's `check` handler finds new versions of the resource.
* A resource prototype's `get` handler produces an output directory named
  `resource`.
* Concourse will `check` periodically to find new versions of the resource.
  * When a job has a `get` step with `trigger: true`, Concourse will
    automatically trigger new builds of the job when new versions of the
    resource are found.
* The `get` step will execute the `get` handler and map its `resource` output
  to the name of the resource, making it available to subsequent steps.
* Concourse will cache the `resource` output so that subsequent builds do not
  have to run the `get` handler on a worker which has already fetched the same
  version.

Removing the syntactic sugar, you could also execute the `get` handler
manually, like so:

```yaml
run: get
type: git
params:
  uri: https://example.com/my-image-src
  ref: v1.5.0
output_mapping:
  resource: my-image-src
```

This will circumvent the caching and auto-triggering and just run the `get`
handler every time the build runs, as if it were a task.


## Out of scope

* [Richer metadata](https://github.com/concourse/concourse/issues/310) - this
  hasn't gained much traction and probably needs more investigation before it
  can be incorporated. This should be easy enough to add as a later RFC.


## Etymology

* The root of the name comes from [Prototype-based
  programming](https://en.wikipedia.org/wiki/Prototype-based_programming), a
  flavor of object-oriented programming, which the concepts introduced in this
  proposal mirror somewhat closely. There are also some hints of actor-based
  programming - actors handling messages and emitting more actors - but that
  felt like more of a stretch.

  This is also an opportunity to frame Concourse as "object-oriented CI/CD," if
  we want to.

* We've been conflating the terms 'resource' and 'resource type' pretty
  everywhere which seems like it would be confusing to newcomers. For example,
  all of our repos are typically named 'X-resource' when really the repo
  provides a resource *type*.

  Adopting a new name which still has the word 'type' in it feels like it fixes
  this problem, while preserving 'resource' terminology for its original use
  case (pipeline resources).

* It's a pun: 'protocol type' => 'prototype'.

* 'Prototype' has a pretty nifty background in aerospace engineering. This is a
  completely different meaning, but we never use it that way, so it doesn't
  seem like there should be much cause for confusion.


## Open Questions

* Is this terminology unrelatable?


[rfc-1]: https://github.com/concourse/rfcs/pull/1
[rfc-1-comment]: https://github.com/concourse/rfcs/pull/1#issuecomment-477749314
[rfc-24]: https://github.com/concourse/rfcs/pull/24
