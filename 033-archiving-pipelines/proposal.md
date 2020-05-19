# Summary

This proposal outlines a relatively straightforward 'archiving' operation which can be used to soft-delete a pipeline while preserving its data for later perusal.

# Proposal

Pipelines can be archived by a user through the `fly` CLI:

```sh
$ fly -t ci archive-pipeline -p pipeline-name
pipeline 'pipeline-name' archived
```

Archived pipelines inherit the behavior of paused pipelines, with a few differences outlined below. As with paused pipelines, no resource checking or job scheduling is performed. Build logs are kept, but remain subject to the configured build log retention policy.

Archived pipelines remain viewable in the web UI and `fly`, but they are grouped into a separate section, hidden by default.

Unlike paused pipelines, archived pipelines will have their configuration stripped out so that sensitive information isn't stored forever.

Unlike paused pipelines, new builds cannot be created for archived pipelines. This is outlined in the API differences below, and enforced by the removal of their configuration.

Archived pipeline names exist in the same namespace as unarchived pipelines. Configuring a new pipeline with the same name as an archived pipeline un-archives the pipeline and gives it a new configuration. See [Un-archiving](#un-archiving).

## API implications

Archived pipelines become read-only, to some extent. API operations that occur within the pipeline, such as triggering jobs and pinning/un-pinning resources, will be rejected. API operations to the pipeline itself behave as follows:

* All pipeline API objects will include a new field, `"archived": true/false`.
* `SaveConfig` (i.e. `fly set-pipeline`, `set_pipeline` step) will work. See [Un-archiving](#un-archiving).
* `ListAllPipelines`, `ListPipelines`, and `GetPipeline` will continue to return archived pipelines. Hiding archived pipelines is the job of the web UI and `fly`, not the API.
* `DeletePipeline` will work, so that an archived pipeline can be removed for good.
* `OrderPipelines` will work. The ordering of a pipeline is pretty unimpactful and users may want to order their archived pipelines too.
* `PausePipeline` will reject the request; archived pipelines are permanently paused.
* `UnpausePipeline` will reject the request; archived pipelines are permanently paused.
* `ExposePipeline` and `HidePipeline` will still work.
* `RenamePipeline` will work; this way an archived pipeline can be named something else so that its original name can be used for unrelated pipelines.
* `GetConfig` will 404; when a pipeline is archived its config is removed to avoid leaking sensitive information. Any other read operation scoped to the pipeline should work.
* `CreateJobBuild` will error; archived pipelines must consume no scheduling resources, not even the build starter.
* `CheckResource` and `CheckResourceType` will error; archived pipelines must consume no scheduling resources, including queuing checks.
* `PinResourceVersion`, `UnpinResource`, `EnableResourceVersion`, `DisableResourceVersion`, and `SetPinCommentOnResource` will error; archived pipelines are read-only. Assume that pins, comments, and enabled/disabled versions persist when a pipeline is unarchived.

## Automatic archiving

Pipelines configured with the [`set_pipeline` step](https://github.com/concourse/rfcs/pull/31) gain additional semantics with regard to archiving.

Given a job with the following build plan:

```yaml
plan:
- set_pipeline: master
- set_pipeline: release-5.0.x
```

When this runs, two pipelines will be created/updated: `master` and `release-5.0.x`.

If a `set_pipeline` step is removed, like so...:

```yaml
plan:
- set_pipeline: master
```

When this runs and the build completes, Concourse will notice that `release-5.0.x` is no longer configured and automatically archive it.

This can be done by keeping track of which job created a pipeline, and which pipelines were produced by each build. When a build completes, Concourse will compare the set of pipelines produced by the build to the set of pipelines associated to its job overall and archive any pipelines not present in the build's set. Alternatively the archiving could be done through a pipeline garbage-collector; there is no guarantee of immediacy.

## Un-archiving

A pipeline will become un-archived when its pipeline is set once again, either through `fly set-pipeline` or through the `set_pipeline` step. This ensures that there is a valid configuration when unarchiving.

Note that when a pipeline is un-archived through `fly set-pipeline`, it will paused, but if a pipeline is un-archived through the `set_pipeline` step, it will be unpaused. This is the same behavior as with a newly configured pipeline.

Because a pipeline becomes un-archived and re-configured in one fell swoop, it's possible that a user may unknowingly "reclaim" an old, archived, unrelated pipeline when really they just want to use the name again for a different pipeline.

To make the behavior explicit, a prompt will be added to `fly set-pipeline` when it detects that the user is configuring an existing but archived pipeline. This is easy to detect, because `fly set-pipeline` already fetches the existing pipeline for diffing purposes. If `fly set-pipeline` is run with `--non-interactive`, the pipeline will be configured and unarchived without a prompt.

The `set_pipeline` step's behavior will be consistent with `fly set-pipeline --non-interactive` as long as the archived pipeline was originally configured by the same job. This way things will "just work" in the happy path of commenting-out and then uncommenting a `set_pipeline` step. If the `set_pipeline` step notices that it's configuring an archived pipeline configured by a *different* job, or by no job, it will fail. The user will have to either rename or destroy the archived pipeline.


# Open Questions

n/a
