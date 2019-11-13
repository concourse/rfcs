# Summary

Introduces `var_sources`, a way to configure multiple named credential
managers, initially at the pipeline-level but potentially at the system-level
and project-level (concourse/rfcs#32).

# Motivation

Concourse currently supports configuring a single credential manager for the
entire cluster. This is limiting in a number of ways.

## Rigid path lookup schemes

With auth for the credential manager configured at the system-level, each
credential manager has to support some form of multi-tenancy so that team A's
secrets are accessed separately from team B's secrets.

The current strategy is to encode team and pipeline names in the paths for the
keys that are looked up, but this has many downsides:

* Path schemes are a brittle approach to the security of untrusted multi-tenant
  installations, as it relies on the credential manager's credential
  identifiers to have a valid separator character that is also not allowed by
  Concourse team and pipeline names. This makes certain credential managers,
  e.g. Azure KeyVault which only allows `[a-z\-]+`, impossible to support.

* This makes it impossible to share credentials between teams. Instead the
  credential has to be duplicated under each team's path. This is a shame
  because credential managers like Vault have full-fledged support for ACLs.

* With Vault, this makes it impossible to use any backend except `kv`, because
  all keys live under the same path scheme, and different backends can't be
  mounted under paths managed by other backends. This removes a lot of the
  value of using Vault in the first place.

## "There can be only one"

Only supporting a single credential manager really limits the possibilities of
using credential managers for specialized use cases.

A core tenent Concourse resources is that their content, i.e. version history
and bits, should be addressable solely by the resource's configuration. That
is, given a resource's `type:` and `source:`, the same version history will be
returned on any Concourse installation, and can therefore be de-duped and
shared across teams within an installation. This means not relying on cluster
state for access control; resource types should entirely trust their `source:`.

This is problematic for resources which make use of IAM roles associated to
their `worker` EC2 instances in order to authenticate, because in this case the
resource's `source:` does not actually include any credentials. As a result, we
cannot safely enable [global resources][global-resources-opt-out]. by default
because these resources would share version history without even vetting their
credentials.

A special credential manager could be implemented to acquire credentials via
IAM roles on the `web` EC2 instance and then provide them to the `source:`
configuration via `((vars))`. This way the `source:` configuration is the
source of truth. This is discussed in [concourse/concourse#3023][issue-3023].

However, as there can only be one credential manager configured at a time,
using that single "slot" just for IAM roles is a bit of a waste compared to a
full-fledged credential manager that can be used for many more things.

# Proposal

This proposal introduces a new kind of configuration: `var_sources`.

This name "var source" is chosen to build on the existing terminology around
`((vars))` and to directly relate them to one another. Calling them "var
sources" instead of "credential managers" also allows them to be used for
things that aren't necessarily credentials.

`var_sources` may be specified at a pipeline-level, like so:

```yaml
var_sources:
- name: vault
  type: vault
  config:
    uri: https://vault.example.com
    # ... vault-specific config including auth/etc ...

resources: # ...

jobs: # ...
```

Each var source has a `name` which must be a [valid
identifier][valid-identifier-rfc]. This is used to explicitly reference the
source from `((vars))` syntax so that there is no ambiguity. See
[`VAR_SOURCE_NAME`](#VAR_SOURCE_NAME).

A var source's `type` specifies one of the supported credential managers, e.g.
`vault`, `credhub`, `kubernetes`. is responsible for interpreting
`config`.

A var source's `config` is a "black box" to Concourse and is passed verbatim to
the credential manager. This configuration should include any credentials
necessary for authenticating with the credential manager.

A var source's `config` may use `((vars))` to obtain its own credentials,
either by using a different var source or by using the system-level credential
manager. These may also be provided statically at pipeline-configuration time.
Either way, the config will be encrypted when stored in the database.

## `((var))` syntax

The `((var))` syntax was introduced a long while back and was never formally
specified or documented. This RFC proposes a change to it so now's a good time
to lay it all out.

The full `((var))` syntax will be
**`((VAR_SOURCE_NAME:SECRET_PATH.SECRET_FIELD))`**.

### `VAR_SOURCE_NAME`

The optional `VAR_SOURCE_NAME` segment specifies which named entry under
`var_sources` to use for the credential lookup. If omitted (along with the
`:`), the globally configured credential manager is used.

A `VAR_SOURCE_NAME` must be a valid identifier per concourse/rfcs#(TODO).

### `SECRET_PATH`

The required `SECRET_PATH` segment specifies the secret to be fetched. This can
either be a single word (`foo`) or a path (`foo/bar` or `/foo/bar`), depending
on what lookup schemes are supported by the credential manager. For example,
Vault and CredHub have path semantics whereas Kubernetes and Azure KeyVault
only support simple names.

For credential managers which support path-based lookup, a `SECRET_PATH`
without a leading `/` may be queried relative to a predefined set of path
prefixes. This is how the Vault credential manager currently works; `foo` will
be queried under `/concourse/(team name)/(pipeline name)/foo`. See [Path lookup
rules](#path-lookup-rules) for more information.

### `SECRET_FIELD`

The optional `SECRET_FIELD` specifies a field on the fetched secret to read. If
omitted, the credential manager may choose to read a 'default field' from the
fetched credential, if it exists. For example, the Vault credential manager
will return the value of the `value` field if present. This is useful for
simple single-value credentials.

## Credential manager secret lookup rules

Pipeline-level credential managers differ from globally-configured credential
managers in one key way: they do not have to be limited to a particular path
scheme.

This means that credentials can be shared between teams, and credential manager
specific settings such as ACLs may be utilized to securely share access to
common credentials.

Credential managers may still choose to have default path lookup schemes for
convenience. This RFC makes no judgment call on this because the utility of
this will vary between credential managers.


# Open Questions

* How should pipeline-level credential managers differ in behavior from
  system-level credential managers, if at all?

* Should we allow multiple var sources to be configured at the system-level?

* Assuming `var_sources` can be configured at the [project][projects-rfc]-level are also be configurable in the future,
  how should they interact with pipeline-level `var_sources`? Should an
  ambiguity error be thrown if the same name is used?

* When and how often do we authenticate with each credential manager? If you're
  using Vault with a periodic token, something will have to continuously renew
  the token.

  Will the `web` node have to maintain long-lived clients for accessing each
  configured credential manager across all teams? Is that going to be a
  scalability concern? Is that going to be a security concern? (Can it be
  avoided?)

# Answered Questions

* n/a

[global-resources-opt-out]: https://concourse-ci.org/global-resources.html#some-resources-should-opt-out
[issue-3023]: https://github.com/concourse/concourse/issues/3023
[projects-rfc]: https://github.com/concourse/rfcs/pull/32
[valid-identifier-rfc]: https://github.com/concourse/rfcs/pull/40
