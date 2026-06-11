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
  // The object to act on.
  Object Object `json:"object"`

  // Path to a file into which the prototype must write its InfoResponse.
  ResponsePath string `json:"response_path"`
}

// InfoResponse is the payload written to the `response_path` in response to an
// InfoRequest.
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
  // The object to act on.
  Object Object `json:"object"`

  // Configuration for establishing TLS connections.
  TLS TLSConfig `json:"tls,omitempty"`

  // A base64-encoded 32-byte encryption key for use with AES-GCM.
  Encryption EncryptionConfig `json:"encryption,omitempty"`

  // Path to a file into which the message handler must write its MessageResponses.
  ResponsePath string `json:"response_path"`
}

// MessageResponse is written to the `response_path` for each object returned
// by the message. Multiple responses may be written to the same file,
// concatenated as a JSON stream.
type MessageResponse struct {
  // The object.
  Object Object `json:"object"`

  // Encrypted fields of the object.
  Encrypted EncryptedObject `json:"encrypted"`

  // Metadata to associate with the object. Shown to the user.
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

type EncryptionConfig struct {
  // The encryption algorithm for the prototype to use.
  //
  // This value will be static, and changing it will imply a major bump to the
  // Prototype protocol version. It is included here as a helpful indicator so
  // that prototype authors don't have to guess at the payload.
  Algorithm string `json:"algorithm"`

  // A base64-encoded 32-length key, unique to each message.
  Key []byte `json:"key"`
}

// EncryptedObject contains an AES-GCM encrypted JSON payload containing
// additional fields of the object.
type EncryptedObject struct {
  // The base64-encoded encrypted payload.
  Payload []byte `json:"payload"`

  // The base64-encrypted nonce.
  Nonce []byte `json:"nonce"`
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

The command must write an `InfoResponse` to the file path specified by
`response_path` in the `InfoRequest`. This response specifies the prototype
interface version that the prototype conforms to, an optional icon to show in
the UI, and the messages supported by the given object.

### Example **info** request/response

Request sent to `stdin`:

```json
{
  "object": {
    "uri": "https://github.com/concourse/concourse"
  },
  "response_path": "../info/response.json"
}
```

Response written to `../info/response.json`:

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


## Encryption

In order to use Prototypes for credential acquisition, there must be a way to
return object attributes which contain sensitive data without writing the data
to disk in plaintext.

A Prototype's `MessageRequest` may contain an `EncryptionConfig` which
specifies the encryption algorithm to use and any other necessary data for use
with the algorithm (such as a key).

The Prototypes protcol will only support one encryption algorithm at a time,
and if it needs to be changed, this will imply a **major** bump to the protocol
version. This is to encourage phasing out support for no-longer-adequate
security algorithms.

The decision as to which algorithm to use for the initial version is currently
an open question, but the most likely candidate right now is AES-GCM, which is
the same algorithm used for database encryption in Concourse. Another candidate
may be NaCL. Note that whichever one we choose must be available in various
languages so that Prototype authors aren't restricted to any particular library
or language.

Assuming AES-GCM, the `EncryptionConfig` in the request will include a `key`
field containing a base64-encoded 32-length key, and a `nonce_size` field
indicating the size of the nonce necessary to encrypt/decrypt:

```json
{
  "object": {
    "uri": "https://vault.example.com",
    "client_token": "01234567889abcdef"
  },
  "encryption": {
    "algorithm": "AES-GCM",
    "key": "aXzsY7eK/Jmn4L36eZSwAisyl6Q4LPFIVSGEE4XH0hA=",
    "nonce_size": 12
  }
}
```


It is the responsibility of the Prototype implementation to generate a nonce
value of the specified length. The Prototype would then marshal a JSON object
containing fields to be encrypted, encrypt it with the key and nonce, and
return the encrypted payload along with the nonce value in a `EncryptedObject`
in the `MessageResponse` - both as base64-encoded values:

```json
{
  "object": {
    "public": "fields"
  },
  "encrypted": {
    "nonce": "6rYKFHXh43khqsVs",
    "payload": "St5pRZumCx75d2x2s3vIjsClUi9DqgnIoG2Slt2RoCvz"
  }
}
```

The encrypted payload above is `{"some":"secret"}`, so the above response
ultimately describes the following object:

```json
{
  "public": "fields",
  "some": "secret"
}
```

### Rationale for encryption technique

A few alternatives to this approach were considered:

* Initially, the use of HTTPS was appealing as it would avoid placing data on
  disk entirely.

  However this is a much more complicated architecture that raises many more
  questions:

  * What are the API endpoints?
  * How are the requests routed?
  * How is TLS configured?
  * How is the response delivered?
    * Regular HTTP response? If the `web` node detaches, how can we re-attach?
    * Callbacks? What happens when the callback endpoint is down?
  * Is it safe to retry requests?

* We could write the responses to `tmpfs` to ensure they only ever exist in
  RAM.

  The main downside is that operators would have to make sure that swap is
  disabled so that data is never written to disk. This seems like a safe
  assumption for Kubernetes but seems like an easy mistake to make for
  operators managing their own VMs.

One advantage of the current approach is that it tells Concourse which fields
can be public and which fields contain sensitive information, while requiring
the sensitive information to be encrypted.

While this could be supported either way by specifying a field such as
`expose":["public"]`, it seems valuable to force Prototype authors to "do the
right thing" and encrypt the sensitive data, rather than allowing them to
simply hide it from the UI.


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

## Pipeline Usage

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

- name: my-image
  type: registry-image
  source:
    repository: example/my-image
    username: some-username
    password: some-password

jobs:
- name: build-and-push
  plan:
  - get: my-image-src
  - run: build
    type: oci-image
    input_mapping: {context: my-image-src}
    outputs: [image]
  - put: my-image
    params: {image: image/image.tar}
```

This example also makes use of resources, which are also implemented as
prototypes with special pipeline pipeline semantics and step syntax. These
"resource prototypes" are described in [RFC #38][rfc-38].


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

### Inputs/Outputs

An open question is how best to provide inputs and outputs to prototypes,
particularly via the `run` step. The mockup of the `run` step in the [Pipeline
Usage](#pipeline-usage) section above suggested these could be configured via
`inputs`/`input_mapping` and `outputs`/`output_mapping`, but this approach has
some downsides:

* Stutter when `run.params` references the inputs, as you need to specify the
  artifact name twice, e.g.
  ```yaml
  run: build
  type: go
  inputs: [my-repo]
  params:
    package: my-repo/cmd/my-cmd
  output_mapping: {binary: my-repo-binary}
  ```
  * In this case, this could be avoided by using an `input_mapping` to a name
    specific to the prototype. However, this only works when the prototype
    takes in a fixed set of inputs with a fixed set of names, but doesn't
    work when the prototype takes in a list of inputs, for instance.
* It's awkward to define a set of outputs that depends on the inputs. For
  instance, imagine a `go` prototype that can compile multiple packages
  simultaneously, and emit an `output` for each one. Under this input/output
  approach, this may look something like:
  ```yaml
  run: build
  type: go
  inputs: [repo1, repo2]
  params:
    packages:
      cmd1-binary: repo1/cmd/cmd1
      cmd2-binary: repo1/cmd/cmd2
      cmd3-binary: repo2/cmd/cmd3
  outputs:
  - cmd1-binary
  - cmd2-binary
  - cmd3-binary
  ```
  or
  ```yaml
  run: build
  type: go
  inputs: [repo1, repo2]
  params:
    packages:
    - repo1/cmd/cmd1
    - repo1/cmd/cmd2
    - repo2/cmd/cmd3
  output_mapping:
    binary1: cmd1-binary
    binary2: cmd2-binary
    binary3: cmd3-binary
  ```
  In the first case, the prototype defines a pseudo-`output_mapping` in its
  config, which requires repetition when defining the set of `outputs`. In the
  second case, the `outputs` repetition is gone, but the prototype needed to
  invent a naming scheme for the outputs (in this case, suffixing a fixed name
  with the 1-based index of the package). Both approaches are fairly awkward to
  work with.

Here are some alternative approaches that have been considered:

#### Option 1a - dynamic input/output config

The prototype's `info` response will include the required
inputs/outputs(/caches?) based on the request object (i.e. `run.params`).

For instance, with the following `run` step:

```yaml
run: some-message
type: some-prototype
params:
  files: [some-artifact-1/some-file, some-artifact-2/some-other-file]
  other_config: here
  output_as: some-output
```

...the prototype may emit the following config, ascribing special meaning to
`files` and `output_as`:

```yaml
inputs:
- name: some-artifact-1
- name: some-artifact-2

outputs:
- name: some-output
```

...and Concourse will mount these artifacts appropriately.

**Pros**

* No new concepts
  * If you're familiar with `task` configs, this is effectively just a more
    flexible version of the same concept
  * No new pipeline syntax/semantics
* Prototype details can be encapsulated behind config
  * e.g. with the [oci-build-task], if you want to persist the build cache, you
    need to specify: `caches: [{path: cache}]`
  * With an approach like this, you could just specify: `cache: true` (i.e. you
    don't need to know where the cache is)

**Cons**

* Requires inventing a naming scheme when the set of outputs is dynamic based
  on the inputs
* Has performance implications. Ignoring caching, `run` steps will need to spin
  up two containers (one for making the `info` request, and one for running the
  prototype message)
  * Caching is possible when the configuration is fixed, but with variable
    interpolation it may not work so well
* More burden on prototype authors as each message now needs two handlers - one
  for executing the message, and one for generating the inputs/outputs.

#### Option 1b - [JSON schema] based dynamic input/output config

Similar to 1a in that we rely on the prototype to instruct Concourse on what
the inputs/outputs are. However, rather than the `info` response providing the
inputs/outputs config, it instead gives a [JSON schema] for each message, with
some special semantics for defining inputs/outputs:

e.g. for a `build` message on an `oci-image` prototype:

```json
{
  "type": "object",
  "properties": {
    "files": {
      "type": "array",
      "items": {
        "type": "string",
        "concourse:input": {
          "name": "((name))"
        }
      }
    },
    "other_config": {
      "type": "string"
    },
    "output_as": {
      "type": "string",
      "concourse:output": {
        "name": "((name))"
      }
    }
  },
  "required": ["context"]
}
```

This isn't fully fleshed out, but the key concept is these `concourse:*`
keywords, which allow you to say: this element in the object is an
input/output.

**Pros**

* Performance implications from 1a disappear - much easier to cache
* All inputs/outputs need to be mentioned in the config, so nothing is implicit
  * Can also be viewed as a Con - more verbose, and is closer to the original
  * Unlike the `inputs`/`input_mapping`, `outputs`/`output_mapping` approach,
    however, this approach requires less repetition, as the inputs/outputs only
    need to be defined in one place
* Gives a way for Concourse to easily validate input to a prototype, and for
  IDE's to provide more useful auto-complete suggestions - related to
  https://github.com/concourse/concourse/issues/481
  * The IDE aspect may not be practical as getting the JSON schema requires
    making the `info` request against a Docker image.

**Cons**

* Makes it much more of a burden to write prototypes, as each message *needs* a
  JSON schema
* Probably too restrictive - requires Concourse to support specific custom
  keywords in the JSON schema. If you wanted to define a map of inputs to
  paths, for instance, we'd need to provide a special keyword for that (as
  `concourse:input` alone isn't enough)

#### Option 2a - emit outputs in response objects

This option implies two things:

1. A way for pipelines to explicitly specify what inputs should be provided to
   a prototype (whereas options 1a/b had the prototypes telling Concourse what
   inputs should be provided)
2. A way for prototypes to provide output artifacts as part of its message
   response (whereas option 1a included in the info response, i.e. not at
   "runtime" w.r.t. running the message)

The response objects could look something like:

```json
{
  "object": {
    "some_data": 123,
    "some_output": {"artifact": "./path"}
  }
}
```

Concourse will interpret `{"artifact": "./path"}` as an output, where `./path`
is relative to some path that's mounted by default to all prototypes. *This
means that multiple outputs may share an output volume, and differ only by the
path within that volume.*

Since prototypes can emit multiple response objects, this also means you can
have *streams* of outputs sharing a name that are identified by some other
field(s). e.g. the above artifact could be identified as
`some_output{some_data: 123}` (or something along those lines). That gives you
the option of aggregating/filtering the stream of outputs by a subset of the
fields that identify them - a similar idea has been fleshed out for the
`across` step in
https://github.com/concourse/rfcs/pull/29#discussion_r619863020.

w.r.t. providing inputs, we can use an approach similar to `put.inputs:
detect`, but more explicit (from the pipeline's perspecive). e.g.

```yaml
run: some-message
type: some-prototype
params:
  files: [@some-artifact-1/some-file, @some-artifact-2/some-other-file]
  other_config: here
```

Here, `@` is a special syntax that points to an artifact name and both *mounts
it to the container* and *resolves to an absolute path to the artifact*. In
this example, the prototype would receive something like:

```json
{
  "object": {
    "files": ["/tmp/build/some-artifact-1/some-file", "/tmp/build/some-artifact-2/some-file"],
    "other_config": "here"
  },
  "response_path": "..."
}
```

Interestingly, data and artifacts are all collocated, which raises the question
- do we need a separate namespace for artifacts like we have now, or can they
be treated as "just vars" and share the local vars namespace?

In order to use emitted artifacts within the pipeline, you can "set" them to
the local namespace:

```yaml
run: some-message
type: some-prototype
params:
  files: [@some-artifact-1/some-file, @some-artifact-2/some-other-file]
  other_config: here
set_artifacts: # or, if we do collapse the namespaces, this could be `set_vars`
  some_output: my-output # some awkwardness around _ vs - here
```

**Pros**

* Removes burden from prototype authors to define what inputs/outputs are
  required up-front
* More flexible - if set of outputs depends on things only determinable at
  runtime, you can express that here
* Output streams let you do interesting filtering (example in
  https://github.com/concourse/rfcs/pull/29#discussion_r619863020), and avoid
  the issue of needing to invent a naming scheme with sets of common outputs.

**Cons**

* New pipeline syntax/concepts to learn
* Can't mount inputs at specific paths - if your prototype requires a specific
  filesystem layout, it needs to shuffle the inputs around
  * e.g. [oci-build-task] may depend on inputs being at certain paths relative to
    the `Dockerfile`
* Since outputs can appear anywhere in the response object, different
  prototypes may provide a different way to interact with outputs, rather than
  having a single flat namespace of outputs
  * Can also be viewed as a Pro, but I feel like having a consistent way of
    referring to artifacts within a pipeline is beneficial

**Questions**

* Is merging the concepts of vars and artifacts confusing/unintuitive?

#### Option 2b - emit outputs adjacent to the response object

This approach uses the same `@` syntax semantics as 2a - the main difference is
that it explicitly differentiates between outputs and data by emitting outputs
in the message response, but not in the response object. e.g.

```json
{
  "object": {
    "some_data": 123
  },
  "outputs": [
    {"name": "some_output", "path": "./path/to/output"}
  ]
}
```

**Pros**

Like 2a, but also:
* Consistent with the existing notion of outputs (i.e. flat namespace of
  artifacts than can be referenced by name)
  * Still allows for filtering the stream down by the corresponding `object`

**Cons**

Like 2a, but also:
* Can't unify concepts of vars and artifacts


[rfc-1]: https://github.com/concourse/rfcs/pull/1
[rfc-1-comment]: https://github.com/concourse/rfcs/pull/1#issuecomment-477749314
[rfc-24]: https://github.com/concourse/rfcs/pull/24
[rfc-38]: https://github.com/concourse/rfcs/pull/38
[oci-build-task]: https://github.com/vito/oci-build-task
[JSON schema]: https://json-schema.org/
