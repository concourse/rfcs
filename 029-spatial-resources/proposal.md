# Spatial Resources

This proposal uses the [generalized resource interface](../024-generalized-resources/proposal.md) to show how the interface would be implemented and interpreted by Concourse to support "spatial resources", which model change over *space*, not time (as with [artifact resources](../024-artifact-resources/proposal.md)).

## Motivation

* Support multi-branch workflows: [concourse/concourse#1172](https://github.com/concourse/concourse/issues/1172)

* Begin phasing out `version: every` by using `across` for batch operations within a single build: [concourse/concourse#736](https://github.com/concourse/concourse/issues/736)

## Proposal

Spatial resources are implemented by having their `check` action yield a 'config fragment' for each space.

Spatial resources are used via the `across` step, which executes a sub-step for each space.

Let's say we have a `supported-go-versions` resource, which is a spatial resource pre-configured with a set of Docker Registry image tags to monitor:

```yaml
type: registry-images
source:
  repository: golang
  tags: [1.10, 1.11]
```

The `registry-images` resource type is similar to the `registry-image` resource type, except instead of monitoring a single tag it monitors *many* tags and emits a config fragment for each tag's current digest. Something like this:

```json
{"fragment":{"tag":"1.10","digest":"sha256:abcdef"}}
{"fragment":{"tag":"1.11","digest":"sha256:abcdef"}}
```

This resource's `get` action would then fetch the digest specified by the fragment.

This can be used with the `across` step to execute a `task` across each of these images:

```yaml
plan:
- across: supported-go-versions
  as: go-image
  do:
  - task: unit
    image: go-image
```

The `across` step would execute the `task` once for each fragment, executing the resource's `get` action for each fragment and producing an artifact called `go-image`. This artifact is then used by the `task` as its image.

## New Implications

### Build matrixes

The `across` step can be nested to form a build matrix:

```yaml
plan:
- across: supported-go-versions
  as: go-image
  do:
  - across: supported-postgres-versions
    as: postgres-image
    do:
    - task: testflight
      image: go-image
```

### Pipeline matrixes

When used with the `set_pipeline` step ([RFC #31](https://github.com/concourse/rfcs/pull/31)) and instanced pipelines ([RFC #34](https://github.com/concourse/rfcs/pull/34)), it can be used to automate spatial pipeline matrixes:

```yaml
plan:
- across: concourse-branches
  as: concourse-branch
  do:
  - set_pipeline: branch
    instance_vars:
      branch: ((concourse-branch.name))
```

(The above example cheats a bit and assumes we've implemented support for using `((vars))` to access fields of config fragments. RFC TBD.)
