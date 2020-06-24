* RFC PR: [concourse/rfcs#31](https://github.com/concourse/rfcs/pull/31)
* Concourse Issue: [concourse/concourse#5814](https://github.com/concourse/concourse/issues/5814)

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

Later RFCs (namely, 'projects' and 'instanced pipelines') will build on this idea to provide a truly continuous workflow for automating pipelines - including their automatic archival when they're no longer needed.


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

Upon first configuration the pipeline will be automatically unpaused, as opposed to `fly set-pipeline` which puts newly configured pipelines in paused state by default. The assumption here is that if you're automating `set_pipeline` you're not just kicking the tires and can probably trust the pipelines that you're configuring are correct, at least enough to have made it into version control.

When configuring an existing pipeline, however, the pipeline's paused status will not be changed. In other words, the `set_pipeline` step will leave already-existing paused pipelines in the paused state. The assumption here is that the pipeline has been manually paused by a pipeline operator, possibly in response to an emergent situation, and it should be left alone.

## `((vars))` support

Additionally, we should support `vars` (as in `fly set-pipeline -y`) and `var_files` (i.e. `fly set-pipeline -l`):

```yaml
plan:
- get: ci
- set_pipeline: release
  file: ci/pipelines/release.yml
  vars: {release_version: 5.3}
  var_files:
  - ci/pipelines/vars/foo.yml
```

## Preventing manual updates

When using `fly set-pipeline` to update a pipeline that has been configured
through the `set_pipeline` step, a warning will be printed and a confirmation
dialogue will be presented.

When configured through `fly set-pipeline` thereafter, warnings will no
longer be issued.

This is to prevent accidentally configuring changes that will be blown away,
while still allowing pipeline operators to take over its configuration if
needed.


# Experiments

There are a few extended pieces of functionality that have been proposed. There
is currently no consensus on these being the ideal long-term design, because
there are alternative methods we're planning that should make them unnecessary.

However, there is value in supporting them "until we get there." We can
implement support for them, and include a warning both in their usage and in
the documentation that they may be removed in the future.

Each experiment must have an easy-to-find GitHub Discussion so that we can
collect feedback on how the feature is used and confirm that the long-term
design addresses the core need appropriately.

## `set_pipeline: self`

* PR: [#4857](https://github.com/concourse/concourse/pull/4857)
* Feedback: [#5732](https://github.com/concourse/concourse/discussions/5732)

Currently, the `foo` in `set_pipeline: foo` is the name of a pipeline to set. A
pipeline could technically update itself by configuring its own name in the
step, but pipeline configs aren't meant to contain their own name, as doing so
prevents the config from being re-used as a 'pipeline template'. You could of
course turn this into a var, but that's a little clunky to use.

To support self-updating pipelines without making them self-aware, we can allow
the keyword `self` to mean the current pipeline. There is precedent for such a
keyword in other fields like `version: every`, `version: latest`, `inputs:
all`, and `inputs: detect`.

One downside of this approach is it doesn't cover the full lifecycle of the
pipeline: who set it initially, so that the `set_pipeline: self` step can even
run?

This is a question that will likely be answered by the [Projects
concept][projects-rfc] once it's introduced. Projects are designed to be the
authoritative source for pipeline configuration, covering both the initial
creation and the later updating of all pipelines contained therein.

As such, it will be a little odd to support both `set_pipeline: self` and
Projects side-by-side. But until Projects lands, there is benefit in allowing
it so that we can confirm that Projects covers all the use cases for it by
analyzing user feedback.

## Setting pipelines in other teams

* PR: [#5729](https://github.com/concourse/concourse/pull/5729)
* Feedback: [#5731](https://github.com/concourse/concourse/discussions/5731)

The `set_pipeline` step is designed to be a "piece of the puzzle" - just like
other steps like `get`, `put`, and `task`.

It is designed to operate against *one* pipeline, in the current team, and in
the current Concourse cluster. This is in contrast to the
[`concourse-pipeline` resource][concourse-pipeline-resource], which supports
setting *many* pipelines across *many* teams within *any* Concourse cluster.

This step is not intended to be a drop-in replacement for the
`concourse-pipeline` resource, but it *is* a goal to deprecate it. However
full deprecation is blocked on further development around the [Projects
concept][projects-rfc] or other ideas that lead towards auto-configuring the
full Concourse cluster.

The `concourse-pipeline` resource provides significant enough burden to
maintainers and users that it is probably wise to expedite its deprecation
without waiting on these farther-off goals. To this end, we can
experimentally support setting pipelines in other teams by configuring a
`team:` field on the step:

```yml
set_pipeline: foo
team: bar
file: ci/foo.yml
```

This must only work if the step is being run by an admin team (i.e. `main`),
making its usage somewhat limited. Once a more suitable replacement arrives
this field can be removed.


# Open Questions

n/a


# Answered Questions

* > Should we support glob expansion in `var_files`?
  >
  > The `concourse-pipeline` resource supports this by just performing glob
  > expansion against its local filesystem. For the `set_pipeline` step, this is
  > a bit more challenging - there *is* no local filesystem. Would we have to
  > implement glob expansion in the Baggageclaim API or something? How easily
  > would this translate to other runtimes?

  This is a question we'll probably have to answer for various different
  steps, so it should probably be addressed outside of this RFC.


# New Implications

## Deprecating `concourse-pipeline` resource

Deprecating the `concourse-pipeline` resource should be the primary goal.

Some of the extended functionality of the resource will not be supported in the name of keeping the `set_pipeline` step design simple and easy to reason about.

For example, the step should only ever configure one pipeline at a time - it should not support the `pipelines:` functionality for configuring a bunch at once.

Similarly, the step should not support fully dynamic configuration (`pipelines_file:`).


[concourse-pipeline-resource]: https://github.com/concourse/concourse-pipeline-resource
[projects-rfc]: https://github.com/concourse/rfcs/pull/32
