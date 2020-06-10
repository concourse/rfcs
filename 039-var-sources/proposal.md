# Summary

Introduces `var_sources`, a way pipelines to configure multiple named
credential managers - and in the future, arbitrary
[Prototype-based][prototypes-rfc] sources of `((vars))`.


# Motivation

Concourse currently supports configuring a single credential manager for the
entire cluster. This is limiting in a number of ways.

## Rigid path lookup schemes

With auth for the credential manager configured at the system-level, each
credential manager has to support some form of multi-tenancy so that team A
cannot access team B's secrets.

The current strategy is to encode team and pipeline names in the
paths/identifiers for the credentials that are looked up, but this has many
downsides:

* Naming schemes are a brittle approach to the security of untrusted
  multi-tenant installations; it relies on the credential manager's
  identifiers to have a valid separator character that is also not allowed by
  Concourse team and pipeline names. This makes it impossible to support
  certain credential managers, e.g. Azure KeyVault which only allows
  `[a-z\-]+`.

* Forcing team names into the credential identifier makes it impossible to
  share credentials between teams. Instead the credential has to be
  duplicated under each team's path. This is a shame because credential
  managers like Vault have full-fledged support for ACLs.

* With Vault, enforcing a path scheme makes it impossible to use any backend
  except `kv` because different backends can't be mounted under paths managed
  by other backends. This removes a lot of the value of using Vault in the
  first place.

## "There can be only one"

Only supporting a single credential manager really limits the possibilities of
using credential managers for specialized use cases.

A core tenent of Concourse's "Resources" concept is that their content, i.e.
version history and bits, should be addressable solely by the resource's
configuration. That is, given a resource's `type:` and `source:`, the same
version history will be returned on any Concourse installation, and can
therefore be de-duped and shared across teams within an installation.

This tenent forbids relying on worker state for access control within a
resource. Instead, resource types should only use their `source:`.

This is problematic for resource types which make use of IAM roles associated
to the worker EC2 instance that they run on in order to authenticate, because
in this case the resource's `source:` does not actually include any
credentials. As a result, we cannot safely enable [global
resources][global-resources-opt-out] by default because these resources would
share version history without even vetting their credentials.

To resolve this issue, a var source could be implemented as a
[Prototype][prototypes-rfc] that acquires credentials via EC2 IAM roles and
then provides them to the `source:` configuration for a resource via
`((vars))`. This way the `source:` configuration is still the source of
truth, and we can still support worker-configured credentials.

Tying back to this proposal, the above approach would be awkward to implement
as a credential manager. With support for only a single credential manager,
users would have to choose between using a general-purpose credential manager
like Vault vs. a specialized use case such as EC2 IAM roles.

If we introduce support for configuring multiple credential managers, and go
beyond that to allowing them to be implemented at runtime via Prototypes, we
can support all kinds of credential acquisition at once.


# Proposal

This proposal introduces a new kind of configuration: var sources.

This name "var source" is chosen to build on the existing terminology around
`((vars))` and to directly relate them to one another.

Calling them "var sources" instead of "credential managers" also allows them
to be used for things that aren't necessarily credentials. [RFC #27][rfc-27]
introduces a way to trigger a job when a var changes, which can be used for
per-job timed interval triggers. [RFC #29][rfc-29] introduces a way to run a
step "across" all vars, which could be used to e.g. set a pipeline for each
pull request.

Var sources are specified at a pipeline-level, like so:

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

Currently, a var source's `type` specifies one of the supported credential
managers, e.g. `vault`, `credhub`, or `kubernetes`. In the future, this will
refer to a [Prototype][prototypes-rfc].

A var source's `config` is a "black box" to Concourse and is passed verbatim
to the credential manager (or prototype). This configuration should include
any credentials necessary for authenticating with the credential manager.

A var source's `config` may use `((vars))` to obtain its own credentials,
either using static templating, the system-level credential manager, or other
var sources (see [Inter-dependent var
sources](#inter-dependent-var-sources)).

## `((var))` syntax

The `((var))` syntax was introduced a long while back and was never formally
specified or documented. This RFC proposes a change to it so now's a good time
to describe a spec.

The full `((var))` syntax will be
**`((VAR_SOURCE_NAME:SECRET_PATH.SECRET_FIELD))`**.

* #### `VAR_SOURCE_NAME`

  The optional `VAR_SOURCE_NAME` segment specifies which named entry under
  `var_sources` to use for the credential lookup. If omitted (along with the
  `:`), the globally configured credential manager is used.

  A `VAR_SOURCE_NAME` must be a [valid identifier][valid-identifier-rfc].

* #### `SECRET_PATH`

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

* #### `SECRET_FIELD`

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

## Inter-dependent var sources

Var source configuration tends to contain credentials, like so:

```yaml
var_sources:
- name: vault
  type: vault
  config:
    uri: https://vault.concourse-ci.org
    client_token: some-client-token
```

Naturally, `((vars))` would be used here so that the credential isn't
hardcoded into the pipeline:

```yaml
var_sources:
- name: vault
  type: vault
  config:
    uri: https://vault.concourse-ci.org
    client_token: ((vault-client-token))
```

Building on this, a var source could also use another var source in order to
obtain its credentials:

```yaml
var_sources:
- name: k8s
  type: k8s
  config: {in_cluster: true}
- name: vault
  type: vault
  config:
    uri: https://vault.concourse-ci.org
    client_token: ((k8s:vault-client-token))
```

There is precedent for this type of behavior in `resource_types`, where one
type can reference another type for its own `type`.

Cycles can be avoided by having a var source 'ignore' itself when resolving its
own config. This is the same way that cycles are handled with `resource_types`.

Take the following example:

```yaml
var_sources:
- name: source-1
  type: source-1
  config: {foo: ((source-2:bar))}
- name: source-2
  type: source-2
  config:
  config: {foo: ((source-1:bar))}
```

In this setup, rather than going into a loop, both var sources would fail to be
configured. The `source-1` var source would fail because it can't find
`source-1` when trying to resolve the config for `source-2`, and vice-versa.


# Open Questions

n/a


# Answered Questions

* > Assuming `var_sources` can be configured at the [project](https://github.com/concourse/rfcs/pull/32)-level in the future, how should they interact with pipeline-level `var_sources`?

  > Should we allow multiple var sources to be configured at the system-level?

  Let's avoid these concerns for the first pass as they just raise more questions around named var scoping and they're not proven necessary at the moment.

* > What var sources can be used within a var source's ((config))?

  See [Inter-dependent var sources](#inter-dependent-var-sources).

* > When and how often do we authenticate with each credential manager? If
  > you're using Vault with a periodic token, something will have to continuously
  > renew the token.

  The implementation maintains an auth loop for each configured var source,
  anonymously identified by their configuration. Var sources that are not used
  for a certain TTL are closed, terminating their auth loop.


[global-resources-opt-out]: https://concourse-ci.org/global-resources.html#some-resources-should-opt-out
[issue-3023]: https://github.com/concourse/concourse/issues/3023
[projects-rfc]: https://github.com/concourse/rfcs/pull/32
[valid-identifier-rfc]: https://github.com/concourse/rfcs/pull/40
[prototypes-rfc]: https://github.com/concourse/rfcs/pull/37
[rfc-27]: https://github.com/concourse/rfcs/pull/27
[rfc-29]: https://github.com/concourse/rfcs/pull/29