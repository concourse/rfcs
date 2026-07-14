* RFC PR: [concourse/rfcs#143](https://github.com/concourse/rfcs/pull/143)
* Related discussion: [concourse discussion #9053](https://github.com/orgs/concourse/discussions/9053)
* Proof of concept: [gbrlcrpntr/concourse, `job-vars-poc`](https://github.com/gbrlcrpntr/concourse/tree/job-vars-poc/poc)

# Job trigger parameters and webhooks

## Summary

This RFC proposes typed, declared variables which may be supplied when a job
is triggered. These **job vars** are available to the build through the local
var source, for example `((.:branch))`.

Job vars have defaults and presentation metadata in pipeline configuration.
An individual build stores only the explicit overrides supplied when it was
created. Manual triggers may supply overrides through the web UI, `fly`, or the
existing create-job-build API.

The RFC also proposes generic JSON trigger webhooks. A webhook declaration can
authenticate a bounded request, filter its JSON payload, map fields from it to
declared job vars, and optionally deduplicate provider retries before creating
a build. The webhook mechanism is deliberately provider-neutral.

The parameter model does not depend on the webhook endpoint. The two parts can
be implemented and reviewed independently: manual triggers establish the job
var model, while webhooks provide one additional source of overrides.

Job vars in this proposal are non-secret build metadata. Secret or shared
variables require additional authorization, redaction, and scoping semantics
and are left to future RFCs.

## Motivation

Resources give Concourse a precise model for automatically discovering and
versioning external state. They are less direct when a person or another
system needs to initiate a build with a value that is not a resource version.

Examples include:

* deploying a particular branch to a temporary environment;
* choosing a test or test group for an exploratory run;
* selecting a deployment mode such as dry-run;
* supplying a numeric scale or timeout for one build; and
* receiving a pull-request webhook and using fields from its payload in the
  resulting build.

Today these workflows tend to require separate pipeline instances, temporary
pipeline configuration, an external parameters resource, or a relay which
rewrites a pipeline before triggering it. Those approaches turn one build's
input into long-lived pipeline or external-resource state and provide no
native trigger form or build-level audit trail.

[Discussion #9053][discussion] raised the same need. A Concourse maintainer
suggested declaring vars on the job, overriding them when manually triggering,
and reading them through `((.:name))`. The discussion also distinguished this
from selecting resource versions: a job may eventually benefit from both, but
free-form job values are not resource versions.

`fly execute` is not an equivalent workflow. It creates a one-off build for a
single task and does not run a configured pipeline job.

This proposal is not a replacement for Concourse's spatial-automation model
for branches and pull requests. A webhook creates one build from one delivered
event; it does not discover the current set of pull requests, reconcile missed
events, create pipeline instances, or provide cross-build `passed` semantics.
Those workflows remain the domain of resources, `across`, `set_pipeline`, and
instanced pipelines.

## Terminology

* A **job var declaration** is a typed entry under a job's `vars` field.
* A **trigger override** is a concrete value explicitly supplied for one
  build.
* An **effective job var** is the trigger override when present, otherwise the
  declaration's current default.
* A **trigger webhook** is a named job-level rule which authenticates, filters,
  and maps a JSON request into trigger overrides.
* A **delivery ID** is an opaque provider-generated request identifier used to
  make retries of one accepted delivery idempotent for a job and webhook.

The term “var” is used to connect this feature to Concourse's existing var
syntax and, specifically, the local var source introduced by [RFC #27][rfc-27].

## Proposal

### Declaring job vars

A job may declare the vars it accepts:

```yaml
jobs:
- name: deploy
  vars:
    branch:
      type: string
      default: main
      description: branch to deploy
    replicas:
      type: number
      default: 1
    dry_run:
      type: boolean
      default: false
    environment:
      type: enum
      options: [development, staging, production]
      required: true

  plan:
  - task: deploy
    config:
      platform: linux
      run:
        path: ./deploy
        args:
        - --branch=((.:branch))
        - --replicas=((.:replicas))
        - --dry-run=((.:dry_run))
        - --environment=((.:environment))
```

The declaration fields are:

| Field | Meaning |
| --- | --- |
| `type` | `string`, `number`, `boolean`, or `enum`; omitted means `string`. |
| `default` | Value used when the build has no explicit override. |
| `description` | Human-oriented help shown by trigger clients. |
| `required` | Manual and webhook triggers must provide a value. |
| `options` | Allowed string values for an `enum`; invalid for other types. |

A declaration cannot be both `required` and have a `default`. Enum
declarations must provide at least one option. Defaults are validated against
the declared type during pipeline validation.

Job var names follow Concourse's existing identifier validation. Overrides
with undeclared names, null values, incorrect types, or enum values outside the
declared options are rejected.

An optional var without a default is absent from the local var source unless
overridden. Referencing it then produces Concourse's normal unresolved-var
error.

### Runtime scope and interpolation

Effective job vars seed the root scope of the build's existing local var
source before the first step runs. A job reads them using the source-qualified
syntax established by RFC #27:

```yaml
((.:branch))
```

This qualification prevents a trigger value from silently shadowing a
credential lookup or a value from another var source. A later step such as
`load_var` may replace a local value using the existing local-scope behavior.
An `across` step creates a child local scope which inherits job vars; an
`across` var with the same name shadows the job var only within that child
scope and produces the existing shadowing warning. The root job var remains
unchanged after the `across` step.

When a typed value is the entire interpolated YAML node, its type is preserved.
When a number or boolean is embedded in a string, it is rendered using its
canonical textual representation. This extends the existing scalar string
interpolation behavior for all local vars, not only trigger values.

The trigger values are intentionally not added to every task's environment.
Pipeline authors decide where values are used by writing `((.:name))`, keeping
the data flow visible in pipeline configuration.

### Storage and default resolution

Each manually or webhook-triggered build stores a JSON object containing only
its explicit trigger overrides. The object is exposed as `trigger_vars` in
build JSON wherever that build's trigger metadata is visible under the
existing build visibility rules.

Defaults are not copied to the build. When execution begins, Concourse reads
the current job configuration and overlays the stored overrides on its current
defaults.

A rerun copies only the original build's explicit overrides. It therefore
preserves an explicitly selected branch while using the current value of any
unoverridden default. This matches the proof of concept and avoids creating a
second persisted representation of pipeline configuration.

The implications of this choice, and the alternative of snapshotting all
effective values, are discussed under [Open Questions](#open-questions).

Automatic builds have no trigger overrides. They use current defaults. If an
automatic build references a required var with no value, interpolation fails
at runtime in the same manner as any other unresolved local var. The merits of
earlier failure are also an open question.

### Manual trigger API

The existing endpoint remains the canonical manual trigger endpoint:

```text
POST /api/v1/teams/:team/pipelines/:pipeline/jobs/:job/builds
```

It accepts an optional JSON body:

```json
{
  "vars": {
    "branch": "feature/parameterized-builds",
    "replicas": 2,
    "dry_run": true,
    "environment": "staging"
  }
}
```

An empty request retains today's behavior and response. A successful request
continues to return HTTP 200 with the created build. Invalid overrides return
HTTP 400 with a user-oriented validation error.

The Go client retains its existing `CreateJobBuild` method and gains an
additive `CreateJobBuildWithVars` method, avoiding a breaking signature change.

### `fly` and web UI

`fly trigger-job` accepts the same conventions used by other `fly` commands:

```sh
fly trigger-job -j pipeline/job \
  -v branch=feature/parameterized-builds \
  -y replicas=2 \
  -y dry_run=true
```

`-v` supplies a string. `-y` parses a YAML scalar, allowing numbers and
booleans to retain their types. The ATC remains responsible for declaration
and type validation.

When a job declares vars, the web trigger action opens a form instead of
immediately creating the build. Controls reflect the declaration:

* text input for strings;
* numeric input for numbers;
* checkbox for booleans; and
* select input for enums.

The form displays descriptions, defaults, and required state. On a build page
the form can start from that build's explicit overrides, and edited overrides
can be cleared so the current defaults apply. Jobs without declarations keep
today's one-click behavior.

### Generic JSON trigger webhooks

A job may declare one or more trigger webhooks:

```yaml
jobs:
- name: build-pull-request
  vars:
    pr_number:
      type: number
      required: true
    head_sha:
      required: true
    head_ref:
      required: true
    base_ref:
      required: true

  trigger_webhooks:
  - name: pull-request
    authentication:
      hmac_sha256:
        secret: ((github-webhook-secret))
        header: X-Hub-Signature-256
        prefix: sha256=
    delivery_id:
      header: X-GitHub-Delivery
    filter:
      action:
        one_of: [opened, reopened, synchronize, ready_for_review]
      pull_request.draft: false
    var_mapping:
      pr_number: number
      head_sha: pull_request.head.sha
      head_ref: pull_request.head.ref
      base_ref: pull_request.base.ref
```

The endpoint is:

```text
POST /api/v1/teams/:team/pipelines/:pipeline/jobs/:job/builds/webhook
    ?name=pull-request
```

Each webhook configures exactly one authentication form. `token` retains the
existing `webhook_token` query convention from resource check webhooks.
Alternatively, `authentication.hmac_sha256` names a signature header, an
optional prefix to strip, and a secret used to verify the hex-encoded
HMAC-SHA256 signature over the exact request body. Tokens and HMAC secrets may
be resolved through the pipeline's configured var sources. The endpoint does
not require a bearer token. If `name` is omitted, the job must have exactly one
configured trigger webhook.

The request body must be a JSON object and is limited to 1 MiB. Filters map
payload dot paths to either an exact JSON value or an explicit
`{one_of: [...]}` condition. Every filter must match. The explicit operator
avoids making all YAML arrays ambiguous between literal array equality and set
membership. `var_mapping` maps declared var names to dot paths in the payload.
A missing mapped field is omitted for an optional var and rejected for a
required var. Extracted values go through the same declaration validation as
manual overrides.

An optional `delivery_id.header` names a request header containing an opaque
provider delivery identifier. Concourse stores that identifier as trigger
metadata and enforces uniqueness for the job and named webhook. Delivery IDs
must be valid UTF-8 and no more than 256 bytes. The first matching delivery
creates a build; a retry with the same identifier returns the original build
without consuming another job build number. Concourse does not infer delivery
identity from mutable payload fields.

Responses are:

| Condition | Response |
| --- | --- |
| Valid authentication, matching filter, valid mapped vars, new delivery | HTTP 201 and the created build. |
| Valid authentication, previously accepted delivery ID | HTTP 200 and the original build. |
| Valid authentication, non-matching filter | HTTP 200 and `{"skipped": true}`. |
| Invalid token or signature | HTTP 401. |
| Missing configured token/delivery header, malformed JSON, invalid mapping or vars | HTTP 400. |
| Unknown job | HTTP 404. |

The raw payload is discarded after filtering and extraction. The build stores
only the resulting explicit overrides, the optional opaque delivery ID, and
its creator as `webhook:<name>`.

Trigger webhooks are delivery-driven, not reconciled inputs. This first version
deduplicates retries only when the pipeline author configures a provider
delivery-ID header. It does not impose ordering across distinct requests, and
a request which never reaches Concourse creates no build. Mapping a branch or
commit from a payload does not turn it into a Concourse resource version. A PR
job should map the immutable head SHA used for checkout as well as human-facing
branch context; checking out only the mutable branch name creates a race. When
the value represents queryable, versioned external state, a resource and its
check webhook remain the precise and recoverable model.

### Security boundary

Job vars are **not secrets**. They are persisted as ordinary build metadata
and may appear in API responses, the web UI, build plans, task arguments, and
logs. Documentation and trigger forms must warn users not to supply
credentials.

Webhook query tokens can appear in proxy and access logs. Operators retaining
that compatibility mode should use HTTPS, dedicated tokens, and query-string
redaction. HMAC-SHA256 authentication avoids placing the secret in the URL,
authenticates the exact body, and uses constant-time digest comparison. It is
generic rather than a claim to implement every provider's signature scheme;
operators must configure the provider's documented header and prefix.

The webhook declaration's token or HMAC secret is not returned by the job
presentation API. It is resolved only while handling a request. Raw webhook
bodies are neither logged nor persisted by this feature.

Authenticating an inbound request must not launch a build or container,
or execute arbitrary prototype code. The PoC uses the same in-process
credential-manager evaluation available to resource check webhooks. If var
sources later require container-backed prototype execution, webhook tokens
will need either a restriction to cheaply resolvable sources or a separately
cached or precomputed authentication design.

### Compatibility and migration

All interfaces are additive:

* pipelines without `vars` or `trigger_webhooks` behave as before;
* the existing empty-body create-job-build request remains valid;
* the successful manual endpoint status remains HTTP 200;
* the existing Go client method retains its signature;
* old builds have no `trigger_vars` or delivery ID; and
* database migrations add nullable trigger JSONB and text columns, with a
  partial uniqueness constraint applying only to configured delivery IDs.

Initial implementation can be reviewed in separate changes for the core
schema/storage/runtime/API, `fly`, web UI, and webhook support. User-facing
documentation should follow through Concourse's separate documentation
contribution process.

## Design principles

### Expressive by being precise

The proposal introduces one narrowly defined concept: non-secret, typed,
build-local values declared by a job and optionally overridden at trigger
time. Values have a visible origin, validated shape, local namespace, and
build-level audit record.

This is distinct from resources, which version external state; pipeline
instance vars, which identify pipeline instances; credentials, which require
redaction and access control; and one-off builds, which do not run a job.

### Versatile by being universal

The job-var model is independent of deployment, test, or source-control
domains. It is usable through every existing manual trigger surface.

Webhooks operate on generic JSON equality, explicit `one_of` membership, path
extraction, HMAC-SHA256 verification, and opaque delivery-ID headers rather
than a GitHub-, GitLab-, or Bitbucket-specific schema. Providers can change
payloads without requiring Concourse releases; pipeline configuration contains
the mapping.

The feature reuses the local var source and normal interpolation instead of
introducing environment injection or a new templating system.

### Safe by being destructible

Declarations and mappings remain in pipeline configuration, which is the
recoverable source of truth. Overrides belong to builds and disappear with
their build history. Raw webhook payloads do not create a second durable state
store. An optional opaque delivery ID is stored only with the build it
identifies and has no independent lifecycle.

Destroying a pipeline removes its declarations, webhook configuration, and
build overrides. Restoring the pipeline config restores all behavior except
historical build metadata, just as for current builds.

Keeping secrets out of the first version avoids presenting unredacted database
state as a credential feature. Shared or secret values will require a design
that preserves Concourse's recoverability and external-source-of-truth model.

Webhook behavior is restored with pipeline configuration, but webhook delivery
history is not an external source of truth and is deliberately not recreated.
Workflows which must converge after missed notifications should use a resource
check webhook, which wakes normal resource checking instead of consuming the
request payload.

## Relationship to existing Concourse work

### Local vars and var steps

[RFC #27][rfc-27] introduced the local var source, `((.:name))`, and specified
that each build has a local scope populated during execution by steps such as
`load_var`. Job vars use the same model rather than adding another interpolation
mechanism: they initialize the build's root local scope before its first step.

RFC #27 also proposed `get_var`, which could trigger a job when a value fetched
from a configured var source changed and then place that value in the local
scope. The two designs differ in origin and lifetime. `get_var` represents a
pulled current value owned by an external var source; a trigger override is an
explicit value supplied by the actor creating one build. Both are intentionally
distinct from versioned resources.

### `across`, `set_pipeline`, and instanced pipelines

[RFC #29][rfc-29] uses child local scopes to execute one plan across a set or
matrix of values. Job vars instead select one value set for one build. The two
compose: an `across` step may read a job var when determining its values, and
its scoped vars may shadow root job vars according to the existing rules.

The spatial-automation design combines `across`, the [`set_pipeline`
step][rfc-31], and [instanced pipelines][rfc-34] to reconcile a set of durable
pipeline instances, for example one per pull request. Job vars do not identify
a pipeline and do not create or archive instances. They are build metadata for
cases where creating durable pipeline state would be disproportionate.

### Var sources and prototypes

[RFC #39][rfc-39] configures named sources for looking up vars, principally
credentials, and leaves broader source types to the prototype model. Job vars
are not another configured var source: they are non-secret values in the
already-defined build-local `.` source. Secret or shared parameters would
cross that boundary and therefore remain future work.

[RFC #38][rfc-38] deliberately left webhooks out of the resource prototype
interface while suggesting that Concourse might map webhooks to resource
checks. This proposal preserves that path for versioned state. Its direct
trigger webhook covers a different case: a delivered invocation whose mapped
values become explicit metadata for one build. Provider-specific verification
or pluggable mapping may still build on prototypes in a later design.

### Resource check webhooks

Concourse's existing resource webhook ignores the request payload and starts
the resource's normal `check`. This [deliberately retains one eventually
consistent external source of truth][resource-webhook-rationale] and avoids
giving a resource type multiple ways to discover versions. That behavior
remains unchanged and should be preferred whenever an event merely indicates
that queryable resource state may have changed.

Trigger webhooks consume mapped payload fields because their purpose is to
carry non-versioned invocation data into a build. Immutable external identity,
such as a PR head SHA, can make the invoked work precise, while the build still
lacks resource checking, version history, `passed` constraints, and missed
event reconciliation. Those limitations are part of the interface rather than
properties claimed from resources.

## Prior art

### GitLab

[GitLab pipeline inputs][gitlab-inputs] provide typed values with descriptions,
defaults, options, and validation at pipeline creation. Inputs are fixed once
initialized. [GitLab CI/CD variables][gitlab-variables] are more flexible:
they can exist at job, project, group, or instance scope, can be protected or
masked, and are exposed as runtime environment variables.

This proposal borrows the useful declaration and manual-trigger concepts but
draws a narrower boundary:

* values are job-scoped rather than pipeline-, group-, or instance-scoped;
* values are referenced explicitly through `((.:name))` rather than injected
  as environment variables;
* only explicit build overrides are persisted; and
* masking, protection, secrets, and shared scopes are not claimed.

GitLab's separation between immutable inputs and mutable/scoped variables is a
useful warning not to make one Concourse mechanism responsible for both public
parameters and credentials.

### GitHub Actions and Jenkins

[GitHub Actions `workflow_dispatch`][github-inputs] supports required values,
defaults, descriptions, and typed manual inputs. [Jenkins Pipeline
parameters][jenkins-parameters] expose string, text, boolean, choice, and
password parameters through a `params` object.

GitHub's [webhook validation guidance][github-webhook-validation] signs the raw
request body with HMAC-SHA256 in `X-Hub-Signature-256`. Its [webhook best
practices][github-webhook-best-practices] recommend using the unique
`X-GitHub-Delivery` value to detect redeliveries. The proposal models the
underlying signature header and delivery identity generically instead of
hard-coding those GitHub names.

These systems confirm that parameterized manual runs are a common CI workflow.
Concourse differs by placing values in its existing local var scope and by
retaining resources as the model for versioned external inputs.

## Alternatives considered

### A built-in manual-parameters resource

This was discussed in #9053. It would require a special resource with UI and
mutation behavior unlike the existing resource interface. A user-entered
value is also not necessarily versioned external state. Declaring values on
the already-triggerable job is more direct.

### Pipeline instances or rewriting pipeline configuration

A relay can create an instance per value combination, interpolate configuration,
and then trigger a job. This works today but creates durable pipeline state for
one build's choice, requires credentials and external infrastructure, and can
accumulate unbounded instances.

### Parameter files loaded by a resource

A task can read a versioned file and use `load_var`. This remains appropriate
when the parameters are real external, versioned state. It does not provide a
native manual form or a direct webhook-to-build path and requires mutating an
external store before triggering.

### A resource check webhook or `get_var`

If a webhook means “the external source may have changed,” Concourse should
continue to ignore its payload, run the resource's normal `check`, and schedule
from discovered versions. This is resilient to duplicate or missed webhook
deliveries because polling and checking reconcile against the source of truth.

A future `get_var` trigger is similarly appropriate for an unversioned value
which can be fetched from a configured var source and compared with its prior
value. Neither model handles an invocation value known only to the caller
without first persisting it in another system. Trigger webhooks address that
narrow case and make the delivery semantics explicit.

### Provider-specific webhook handlers

Native handlers could validate provider signatures and offer richer event
semantics, but every provider and payload revision would expand Concourse's
maintenance surface. Generic JSON mapping, HMAC-SHA256 verification, explicit
event membership, and configurable delivery-ID headers cover the common PR
webhook mechanics without embedding provider payloads. Provider adapters can
still be considered separately for signature schemes or event semantics which
cannot be expressed safely by this base.

### Snapshotting every effective value

Concourse could resolve and persist defaults when the build is created. This
would make a delayed build or rerun reproduce the old effective parameter set,
but duplicates pipeline configuration into build state and requires every
automatic build creation path to materialize values. This is the principal
open design choice below.

## Open questions

### Should defaults be resolved from current config or snapshotted per build?

The PoC stores only explicit overrides and resolves defaults from current job
configuration when execution starts. Reruns copy only overrides. This is
simple, keeps defaults in pipeline configuration, and resembles other
Concourse runtime configuration lookups.

This also follows the [reasoning recorded during `get_var`
design][get-var-current-values]: resources are for values which must be
versioned and reproduced, while vars may represent a current value which should
be resolved again on rerun. Job vars add one qualification to that precedent
by preserving values the triggering actor explicitly selected while continuing
to treat unoverridden defaults as current configuration.

The tradeoff is that a queued build or rerun can observe a default different
from the one visible when its original build was created. Snapshotting all
effective values would provide stricter replay but would make defaults part of
historical build state and add resolution/migration complexity.

The RFC should settle which notion of reproducibility is expected:

1. reproduce explicit operator intent while current pipeline defaults remain
   authoritative; or
2. reproduce the entire effective parameter set from build creation time.

The PoC intentionally demonstrates option 1 until this discussion reaches
consensus.

### How should required vars interact with automatic builds?

Manual and webhook triggers can reject a missing required value immediately.
Resource-triggered builds have no actor or payload from which to obtain one.
The PoC permits scheduling and produces a normal unresolved-var error only if
the plan references the missing value.

Alternatives include rejecting pipeline configurations where an automatically
triggerable job has a required var without a default, or failing such builds
before plan execution. Both require a precise definition of “automatically
triggerable” and may make otherwise valid manual-only jobs harder to express.

### Are `vars`, `inputs`, or `parameters` the clearest public names?

`vars` connects directly to Concourse interpolation and RFC #27. “Inputs” may
be confused with resource inputs, while “parameters” is more familiar in other
CI systems. The PoC uses job `vars`, request `vars`, and build `trigger_vars`.

### Are query-token and generic HMAC authentication the right initial boundary?

The PoC retains query tokens for consistency with resource webhooks and adds
configurable HMAC-SHA256 verification for providers such as GitHub. Review
should decide whether both modes should launch, whether signed bodies should
be required for new trigger webhooks, and whether header bearer tokens are a
useful generic middle ground. Provider-specific signature canonicalization or
key-discovery schemes should not be added implicitly to the JSON mapper.

## Future extensions

The following are related but explicitly outside this RFC's first
implementation:

* **Secret trigger values.** These need field-level visibility rules,
  redaction, encrypted or external persistence, log handling, and integration
  with `var_sources` or credential managers. Merely adding `secret: true` to
  the current JSONB metadata would be unsafe.
* **Shared variable scopes.** Pipeline-, team-, project-, or organization-level
  values need explicit inheritance, precedence, ownership, and authorization.
  GitLab group/project variables demonstrate both their utility and the
  complexity they introduce.
* **Protection policies.** Restrictions by role, branch, environment, or
  pipeline exposure should be designed alongside shared and secret scopes.
* **Additional validation and value types.** Regex validation, file values,
  arrays, and structured objects can be considered when concrete workflows
  require them.
* **Forwarding values.** Passing selected trigger values to downstream builds
  or `set_pipeline` requires an explicit data-flow and authorization model.
* **Resource-backed choices.** A future UI could combine job vars with resource
  version selection, as suggested in discussion #9053.
* **Additional signature schemes.** Provider-specific canonicalization,
  asymmetric signatures, rotating keys, or a pluggable verifier may build on
  the generic HMAC-SHA256 model.
* **Correlated superseding.** A future declaration could map a stable logical
  key such as PR number and request that older pending or running builds for
  that key be superseded when a newer head SHA arrives. This is intentionally
  not part of the PoC: it requires scheduler and cancellation semantics,
  authorization and audit behavior, race handling, and a decision about
  whether “supersede” means abort, skip, or merely deprioritize. Delivery-ID
  deduplication is narrower: it suppresses retries of the same event and never
  treats distinct PR updates as interchangeable.

## New implications

* Pipeline authors gain a native manual-input workflow and may consolidate
  temporary or nearly identical pipelines. Maintainers should continue to
  discourage large parameter matrices where pipeline instances or separate
  jobs express the workflow more precisely.
* Build metadata can contain operator-supplied values. Operators and users
  must treat them as public within the build's existing visibility boundary.
* Changing a job default can affect queued builds and reruns under the PoC's
  proposed resolution model. The web UI should make current-default behavior
  explicit when resetting or reusing values.
* Generic webhooks allow external systems to create builds without a bearer
  token. Operators must manage dedicated query tokens or signing secrets and
  protect their ingress path.
* Generic webhooks are event-delivery mechanisms. Configured delivery IDs
  suppress exact retries, but distinct updates are not ordered or superseded,
  and missed requests are not reconciled.
* Supporting typed values embedded in strings broadens scalar interpolation
  for local vars and should be covered by compatibility tests.

[discussion]: https://github.com/orgs/concourse/discussions/9053
[rfc-27]: https://github.com/concourse/rfcs/pull/27
[rfc-29]: https://github.com/concourse/rfcs/pull/29
[rfc-31]: https://github.com/concourse/rfcs/pull/31
[rfc-34]: https://github.com/concourse/rfcs/pull/34
[rfc-38]: https://github.com/concourse/rfcs/pull/38
[rfc-39]: https://github.com/concourse/rfcs/pull/39
[get-var-current-values]: https://github.com/concourse/concourse/issues/5815#issuecomment-711415300
[resource-webhook-rationale]: https://github.com/concourse/concourse/issues/331#issuecomment-382419366
[gitlab-inputs]: https://docs.gitlab.com/ci/inputs/
[gitlab-variables]: https://docs.gitlab.com/ci/variables/
[github-inputs]: https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax#onworkflow_dispatchinputs
[github-webhook-validation]: https://docs.github.com/en/webhooks/using-webhooks/validating-webhook-deliveries
[github-webhook-best-practices]: https://docs.github.com/en/webhooks/using-webhooks/best-practices-for-using-webhooks
[jenkins-parameters]: https://www.jenkins.io/doc/book/pipeline/syntax/#parameters
