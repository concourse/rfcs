# Summary

This RFC proposes a new `set_pipeline` step type for configuring pipelines within a build plan.


# Motivation

## Short-term motivation

Lots of folks are already using the [`concourse-pipeline` resource](https://github.com/concourse/concourse-pipeline-resource), however the resource has two fatal flaws:

* Users have to configure a local auth user and pass it to the resource definition.
* The resource is versioned independently of the user's Concourse, meaning the `fly` version won't always be in sync. The resource makes an attempt to resolve this by doing a `sync` after logging in, but this is a pretty clunky problem regardless.

If we had native support for a `set_pipeline` step, both of these problems would go away.

## Long-term motivation

By having a `set_pipeline` step in the build plan, we can start to improve Concourse's story around automating the full CI stack for projects of all sizes. Users can start to trust that pipelines are always configured via CI, and they can go over the build history to see who changed what and when.

Later RFCs (namely, 'projects' and 'instanced pipelines') will build on this idea to provide a truly continuous workflow for automating pipelines - including their automatic archival when they're no longer needed, in the case of instanced pipelines.


# Proposal

Using the step would look something like this:

```yaml
plan:
- get: ci
- set_pipeline: concourse
  file: ci/pipelines/concourse.yml
```

The `x` in `set_pipeline: x` is the pipeline name, and `file:` would be used to specify the pipeline config.

The pipeline would be configured within whichever team the build execution belongs to.

The pipeline would be automatically unpaused, as opposed to `fly set-pipeline` which pauses pipelines by default. The assumption here is that if you're automating `set_pipeline` you're not just kicking the tires and can probably trust the pipelines that you're configuring are correct, at least enough to have made it into version control.

## `((vars))` support

Additionally, we should support `vars` (as in `fly set-pipeline -y`) and `vars_files` (i.e. `fly set-pipeline -l`):

```yaml
plan:
- get: ci
- set_pipeline: release
  file: ci/pipelines/release.yml
  vars: {release_version: 5.3}
  vars_files:
  - ci/pipelines/vars/foo.yml
```

# Open Questions

none


# Answered Questions

none


# New Implications

## Deprecating `concourse-pipeline` resource
Deprecating the `concourse-pipeline` resource should be the primary goal.

Some of the extended functionality of the resource will not be supported in the name of keeping the `set_pipeline` step design simple and easy to reason about.

For example, the step should only ever configure one pipeline at a time - it should not support the `pipelines:` functionality for configuring a bunch at once.

Similarly, the step should not support fully dynamic configuration (`pipelines_file:`).
