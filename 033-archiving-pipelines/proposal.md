# Summary

This proposal outlines a relatively straightforward 'archiving' operation which can be used to soft-delete a pipeline while preserving its data for later perusal.

# Proposal

Pipelines can be archived by a user through the `fly` CLI:

```sh
$ fly -t ci archive-pipeline -p pipeline-name
pipeline 'pipeline-name' archived
```

Archived pipelines are permanently paused - no resource checking or job scheduling is performed. They should consume no scheduling resources - only database space for the build history.

Archived pipelines may have their configuration stripped out, so that old credentials aren't stored forever.

Archived pipelines will be viewable in the web UI, but grouped into a separate section, hidden by default.

Archived pipeline names exist in the same namespace as unarchived pipelines.

Archived pipelines become read-only, and allow only the following operations:

* `fly rename-pipeline`, so that an archived pipeline can be renamed to "reclaim" its name if desired
* `fly destroy-pipeline`, so that an archived pipeline can be removed "for good"

Any web UI or `fly` interactions will fail at the API layer. Web UI elements may be hidden or disabled for archived pipelines.

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

A pipeline will become un-archived when its pipeline is set once again, either through `fly set-pipeline` or through the `set_pipeline` step. This way we can ensure that there is a valid configuration when unarchiving (which allows us to clear out configuration and other unnecessary data when the pipeline is archived).

This is also to support the use case of temporarily commenting out a `set_pipeline` step and then un-commenting it to bring it back.

# Open Questions

n/a
