# Summary

Instanced pipelines group together pipelines which share a common template configured with different ((vars)). They provide a simple two-level hierarchy and automatic archiving of instances which are no longer needed.

# Proposal

Pipelines can be configured with 'instance vars' like so:

```yaml
fly set-pipeline -p branch --instance-var branch=feature/foo
```

This will configure a `branch` pipeline, with the `((branch))` var set to `"feature/foo"`.

Instance vars are used as part of the pipeline identifier in the UI and API. There can be multiple instances of a pipeline with the same name:

```yaml
fly set-pipeline -p branch --instance-var branch=feature/foo
fly set-pipeline -p branch --instance-var branch=feature/bar
```

Instanced pipelines sharing the same name will be grouped together in the web UI.

An individual instance of a pipeline can be manually destroyed, paused, and archived ([RFC #33](https://github.com/concourse/rfcs/pull/33)):

```yaml
fly destroy-pipeline -p branch -i branch:feature/foo
fly pause-pipeline   -p branch -i branch:feature/foo
fly archive-pipeline -p branch -i branch:feature/foo
```

(Side note: `:` vs. `=` is a little weird but it's consistent with `fly check-resource` - we use `=` for assignment and `:` for partial filtering.)

## Automatic archival

When used with the `set_pipeline` step ([RFC #31](https://github.com/concourse/rfcs/pull/31)), instanced pipelines take on additional behavior.

At the end of a build which uses `set_pipeline`, all instances of the named pipelines which were not configured by the build will be automatically archived.

For example, say I have a job which I use to automatically configure my pipelines:

```yaml
plan:
- get: ci
- set_pipeline: release
  file: ci/pipelines/release.yml
  instance_vars:
    version: 5.3
- set_pipeline: release
  file: ci/pipelines/release.yml
  instance_vars:
    version: 5.2
```

Let's say I ship a `5.5` version. Assuming my policy is to only support the last 2 versions, I would update the config like so:


```yaml
plan:
- get: ci
- set_pipeline: release
  file: ci/pipelines/release.yml
  instance_vars:
    version: 5.4
- set_pipeline: release
  file: ci/pipelines/release.yml
  instance_vars:
    version: 5.3
```

When this build runs, the `version: 5.2` instance will be automatically archived.

# New Implications

This functionality will be more and more useful as we expand Concourse's vocabulary to support pipeline automation. Spatial resources ([RFC #26](https://github.com/concourse/rfcs/pull/26)), for example, can be used to automatically configure a pipeline for each branch or PR. When the branch or PR goes away, their pipeline instances will be archived automatically.
