# `across` step

This proposal introduces a mechanism by which a build plan can be executed
multiple times with different `((vars))`, in service of build matxies, pipeline
matrixes, and - building on `var_sources` ([RFC #39][var-sources-rfc]) -
dynamic variations of both.


## Motivation

* Support build matrixes and pipeline matrixes.
* Support multi-branch workflows: [concourse/concourse#1172](https://github.com/concourse/concourse/issues/1172)
* Replace a common use case of `version: every`, which has proven to be very complicated to support.


## Proposal

The `across` step is given a set of values, a name for a `((var))`, and a
sub-plan to execute with each value assigned as the named var.

The set of values can be static, as below:

```yaml
across: [1.12, 1.13]
as: go-version
do:
- task: unit
  vars: {go_version: ((go_version))}
```

In keeping with `((var))` semantics, values may also be complex types.

Their fields can be accessed off the var or passed verbatim, as in the
following examples:

```yaml
across:
- from: 3.0
  to: 5.0
- from: 4.0
  to: 5.0
as: upgrade_path
do:
- set_pipeline: upgrade-test
  instance_vars:
    upgrade_from: ((upgrade_path.from))
    upgrade_to: ((upgrade_path.to))
```

```yaml
across:
- upgrade_from: 3.0
  upgrade_to: 5.0
- upgrade_from: 4.0
  upgrade_to: 5.0
as: upgrade_path
do:
- set_pipeline: upgrade-test
  instance_vars: ((upgrade_path))
```

### Dynamically iterating over `((vars))` from `var_sources`

Instead of static values, a [`((var))` source][var-sources-rfc] may be
referenced by name in order to execute the plan across dynamic `((vars))` at
runtime.

This gives full control of parameterization to the user. For example, a
`github-prs` prototype could be implemented which returns a `((var))` for each
pull request, or the `git` prototype could be used as a var source to provide
each branch as a `((var))`.

```yaml
var_sources:
- name: booklit-prs
  type: github-prs
  config:
    repository: vito/booklit
    access_token: # ...

plan:
- across: booklit-prs
  as: pr
  do:
  - set_pipeline: pr
    instance_vars: {pr_number: ((pr.number))}
```

### Triggering on changes

With `trigger: true` configured, the build plan will run on any change to the
set of `((vars))` - i.e. when a var is added or removed.


## Open Questions

* It seems like triggering could work in both the static and dynamic cases, but
  it's kind of interesting to reason about the static case. Why/why not?

[var-sources-rfc]: https://github.com/concourse/rfcs/pull/39
