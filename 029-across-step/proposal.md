# `across` step

This proposal introduces a mechanism by which a build plan can be executed
multiple times with different `((vars))`, in service of build matxies, pipeline
matrixes, and - building on [`((var))` sources][var-sources-rfc] - dynamic
variations of both.


## Motivation

* Support dynamic multi-branch workflows:
  [concourse/concourse#1172][multi-branch-issue]
* Support build matrixes and pipeline matrixes.
* Replace a common, and very flawed, usage pattern of `version: every`, which
  has proven to be very complicated to support.


## Proposal

The `across` step modifier is given a list containing vars and associated
values to set when executing the step. The step will be executed across all
combinations of var values.

### Static var values

The set of values can be static, as below:

```yaml
task: unit
vars: {go_version: ((.:go_version))}
across:
- var: go_version
  values: [1.12, 1.13]
```

This will run the `unit` task twice: once with `go_version` set to `1.12`, and
again with `1.13`.

### Dynamic var values from a var source

Rather than static values, var values may be pulled from a var source
dynamically:

```yaml
var_sources:
- name: booklit-prs
  type: github-prs
  config:
    repository: vito/booklit
    access_token: # ...

plan:
- set_pipeline: pr
  instance_vars: {pr_number: ((.:pr.number))}
  across:
  - var: pr
    source: booklit-prs
```

The above example will run the `set_pipeline` step across the set of all GitHub
PRs, returned through a `list` operation on the `booklit-prs` var source.

### Running across a matrix of var values

Multiple vars may be listed to form a matrix:

```yaml
set_pipeline: pr-go
instance_vars:
  pr_number: ((.:pr.number))
  go_version: ((.:go_version))
across:
- var: pr
  source: booklit-prs
- var: go_version
  values: [1.12, 1.13]
```

This will run 2 * (# of PRs) `set_pipeline` steps, setting two pipelines per
PR: one for Go 1.12, and one for Go 1.13.

### Controlling parallelism with `max_in_flight`

By default, the steps are executed serially to prevent surprising load
increases from a dynamic var source suddenly returning a ton of values.

To run steps in parallel, a `max_in_flight` must be specified as either `all`
or a number - its default is `1`. Note: this value is specified on each `var`,
rather than the entire step.

With `max_in_flight: all`, no limit on parallelism will be enforced. This would
be typical for when a small, static set of values is specified, and it would be
annoying to keep the number in sync with the set:

```yaml
task: unit
vars: {go-version: ((.:go-version))}
across:
- var: go-version
  values: [1.12, 1.13]
  max_in_flight: all
```

With `max_in_flight: 3`, a maximum of 3 var values would be executed in
parallel. This would be typically set for values coming from a var source,
which may change at any time, or especially large static values.

```yaml
set_pipeline: pr
instance_vars: {pr_number: ((.:pr.number))}
across:
- var: pr
  source: booklit-prs
  max_in_flight: 3
```

When multiple `max_in_flight` values are configured, they are multiplicative,
building on the concurrency of previously listed vars:

```yaml
set_pipeline: pr
instance_vars:
  pr_number: ((.:pr.number))
  go_version: ((.:go_version))
across:
- var: pr
  source: booklit-prs
  max_in_flight: 3
- var: go_version
  values: [1.12, 1.13]
  max_in_flight: all
```

This will run 6 `set_pipeline` steps at a time, focusing on 3 PRs and setting
Go 1.12 and Go 1.13 pipelines for each in parallel.

Note that setting a `max_in_flight` on a single `var` while leaving the rest as
their default (`1`) effectively sets an overall max-in-flight.

### Triggering on changes

With `trigger: true` configured on a var, the build plan will run on any change
to the set of vars - i.e. when a var value is added, removed, or changes.

```yaml
var_sources:
- name: booklit-prs
  type: github-prs
  config:
    repository: vito/booklit
    access_token: # ...

plan:
- set_pipeline: pr
  instance_vars: {pr_number: ((.:pr.number))}
  across:
  - var: pr
    source: booklit-prs
    trigger: true
```

Note that this can be applied to either static `values:` or dynamic vars from
`source:` - both cases just boil down to a comparison against the previous
build's set of values.

### Modifier syntax precedence

The `across` step is a *modifier*, meaning it is attached to another step.
Other examples of modifiers are `timeout`, `attempts`, `ensure`, and the `on_*`
family of hooks.

In terms of precedence, `across` would bind more tightly than `ensure` and
`on_*` hooks, but less tightly than `across` and `timeout`. This seems to be
the most sensible order, so that `attempts` doesn't retry the entire matrix and
`timeout` can be enforced on each step (though the timeout enforcement would
probably work just as well either way).

```yaml
task: unit
timeout: 1h # interrupt the task after 1 hour
attempts: 3 # attempt the task 3 times
across:
- var: go_version
  values: [1.12, 1.13]
on_failure: # do something after all steps complete and at least one failed
```

To apply `ensure` and `on_*` hooks to the nested step, rather than the `across`
step modifier, the `do:` step may be utilized:

```yaml
do:
- task: unit
  on_failure: # runs after each individual step completes and fails
across:
- var: go_version
  values: [1.12, 1.13]
on_failure: # runs after all steps complete and at least one failed
```

This can be rewritten in a slightly more readable syntax by placing the `do:`
below the `across:`:

```yaml
across:
- var: go_version
  values: [1.12, 1.13]
do:
- task: unit
  on_failure: # runs after each individual step completes and fails
on_failure: # runs after all steps complete and at least one failed
```

### Failing fast

With `fail_fast: true` applied to the `across` step, all steps will be
interrupted in the event that one fails:

```yaml
task: unit
timeout: 1h # interrupt the task after 1 hour
across:
- var: go_version
  values: [1.12, 1.13]
fail_fast: true
```

Note: this is the first time a step *modifier* has had additional sibling
fields. In the event of a conflict (e.g. pretending `in_parallel` has
`fail_fast`), the above `do:` syntax may be utilized as a work-around.


## Open Questions

* n/a


## New Implications

* Using `across` with var sources implies the addition of a `list` action for
  listing the vars from a var source. We could build on this to show the list
  of available vars in the UI, which would really help with troubleshooting
  credential manager access and knowing what vars are available.

  Obviously we wouldn't want to show credential values, so `list` should only
  include safe things like credential paths.

* Combining the [`set_pipeline` step][set-pipeline-rfc], [`((var))`
  sources][var-sources-rfc], [instanced pipelines][instanced-pipelines-rfc],
  [pipeline archiving][pipeline-archiving-rfc], and the `across` step can be
  used for end-to-end automation of pipelines for branches and pull requests:

  ```yaml
  set_pipeline: pr
  instance_vars: {pr_number: ((.:pr.number))}
  across:
  - var: pr
    source: prs
    trigger: true
  ```

[set-pipeline-rfc]: https://github.com/concourse/rfcs/pull/31
[instanced-pipelines-rfc]: https://github.com/concourse/rfcs/pull/34
[pipeline-archiving-rfc]: https://github.com/concourse/rfcs/pull/33
[var-sources-rfc]: https://github.com/concourse/rfcs/pull/39
[multi-branch-issue]: https://github.com/concourse/concourse/issues/1172
