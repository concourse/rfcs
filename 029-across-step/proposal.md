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

* Using `across` with a `((var))` source implies that var sources have some
  method for listing the available vars. It would be great to show them in the
  UI, including metadata - similar to how we show the version history of a
  resource.

  With `((var))` sources also being used for credential management, it could be
  kind of scary to make it so easy to perform batch operations across all your
  credentials. Are there use cases for that? Automating a migration from one
  credential manager to another? Automated credential scanning of some sort?

  It may be the case that credential manager prototypes just don't implement a
  `list` action and have no use for the `across` step. It's not necessary for
  their traditional use in `((var))` syntax, which explicitly names the var to
  fetch. So it could be an optionally supported message, just like how
  resources optionally support `put`.

  Listing credentials could aid in discoverability and debugging failed
  credential fetches, though we obviously wouldn't want to show the credential
  values themsleves on this page, or have them returned by `list`.

* In order to reduce usage of `version: every` with the eventual goal of
  deprecating it, should we have some way of going "across" versions of a
  resource? This could be a better way to represent batch operations and
  running full pipelines per-commit.

  ```yaml
  across: my-repo
  as: commit
  do:
  - set_pipeline: each-commit
    instance_vars:
      commit: ((commit.ref))
  ```


## New Implications

* Combining the [`set_pipeline` step][set-pipeline-rfc], [`((var))`
  sources][var-sources-rfc], [instanced pipelines][instanced-pipelines-rfc],
  [pipeline archiving][pipeline-archiving-rfc], and the `across` step can be
  used for end-to-end automation of pipelines for branches and pull requests:

  ```yaml
  across: prs
  as: pr
  trigger: true
  do:
  - set_pipeline: pr
    instance_vars: {pr_number: ((pr.number))}
  ```

* Nesting the `across` step results in a fairly intuitive build matrix:

  ```yaml
  across: [1.11, 1.12]
  as: golang_version
  do:
  - across: [linux, darwin, windows]
    as: platform
    do:
    - task: build
      vars:
        platform: ((platform))
        golang_version: ((golang_version))
  ```

  Throwing in the `set_pipeline` step and using
  [`instance_vars`][instanced-pipelines-rfc] naturally leads to pipeline
  matrixes:

  ```yaml
  across: [1.11, 1.12]
  as: golang_version
  do:
  - across: [linux, darwin, windows]
    as: platform
    do:
    - set_pipeline: build-and-test
      instance_vars:
        platform: ((platform))
        golang_version: ((golang_version))
  ```


[set-pipeline-rfc]: https://github.com/concourse/rfcs/pull/31
[instanced-pipelines-rfc]: https://github.com/concourse/rfcs/pull/34
[pipeline-archiving-rfc]: https://github.com/concourse/rfcs/pull/33
[var-sources-rfc]: https://github.com/concourse/rfcs/pull/39
[multi-branch-issue]: https://github.com/concourse/concourse/issues/1172
