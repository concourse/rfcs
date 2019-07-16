# Projects RFC

# Summary

A Concourse project is a namespace for resources, tasks, and pipelines. Projects are bootstrapped from a single resource definition and a build plan which executes when new versions of the resource are found. Projects bridge the gap between small-scale and large-scale CI workflows, helping to smoothen Concourse's learning curve.

# Motivation

* Smoothen the learning curve. Right now the jump from `fly execute` to `fly set-pipeline` is pretty steep. Projects can be a user's first introduction to build plans, which transitions nicely into jobs and pipelines.
* Enable small-scale projects to have a simple Travis/Circle CI single-build workflow, without needing pipelines at all. This is a scenario where Concourse pipelines can be seen as overkill.
* Enable medium-to-large-scale projects to fully automate their CI stack by using the project's build plan to configure pipelines via the [`set_pipeline` step](https://github.com/concourse/rfcs/pull/31).
* Enable cross-pipeline `passed` constraints between pipelines within the project. This will allow large pipelines to be broken up into reusable templates while still supporting fan-in downstream.
* Ensure tasks and pipelines are always in sync by having both of them come from the project instead of passing task configs as a resource throughout the pipeline.

# Proposal

A Concourse project allows you to bootstrap and automate your CI/CD stack by configuring a single resource which contains your project's config:

```sh
fly set-project \
  --project booklit \
  --type git \
  --source uri=https://github.com/vito/booklit \
  --source branch=projects-mock \
  --path ci
```

The resource must contain a project config as `project.yml` under the configured `--path`.

This path also contains resource, resource type, task, and pipeline definitions for the project. A full project config directory may look something like this:

```sh
ci/project.yml
ci/resource-types/github-status.yml
ci/resources/booklit.yml
ci/resources/booklit-status.yml
ci/tasks/test.yml
ci/tasks/build.yml
ci/pipelines/booklit.yml
```

## Anatomy of a `project.yml`

A minimal project config has a `name` and a `plan`:

```yaml
name: ci
plan: # ...
```

The `name` should be short and sweet - similar to a resource name. The project config directory will be available to every build executed within the project, as an artifact named after this value. ([Why only the config directory?](#exposing-only-the-config-directory))

Concourse will continuously check the project resource for new versions and execute the project's `plan` when a new version is found. All pipelines, tasks, resources, and resource types defined in the project will be usable by the project's `plan`.

Within the `plan`, you may choose to just run a task, using projects as a simple pipeline-less CI flow:

```yaml
name: ci
plan:
- get: booklit
- task: test
```

...or you may choose to leverage the `set_pipeline` step to automate your pipeline configuration:

```yaml
name: ci
plan:
- set_pipeline: booklit
```

New project builds can be triggered by `get` steps with `trigger: true`, however `get` steps cannot specify any `passed` constraints:

```yaml
name: ci
plan:
- get: booklit
- get: golang
  trigger: true
- task: test
  image: golang
```

If your pipelines use templating, you can automate that just by using a `task`:

```yaml
name: ci
plan:
- task: generate-pipeline
- set_pipeline: my-pipeline
  file: generated-pipeline/pipeline.yml
```

This versatility should help smoothen Concourse's learning curve - there now exists a simpler CI/CD workflow suitable for side-projects and simple workflows that don't really need pipelines at all, and along the way we've enabled arbitrary automation needs for more complex workflows.

## Project-level resources

Projects define resources under the `resources/` directory. These resources can be used by all pipelines within the project.

Resources defined at the project level can be propagated between pipelines using `passed` constraints. This is not possible with pipeline-local resource definitions.

## Pipeline-local resources

Pipelines can still define their own resources in addition to the project scope.

There is some risk for confusion as developers will have two places to look for resource definitions, but the benefits seem to outweigh the risk:

* Pipeline-local resources can be used to prevent polluting the project-wide resource scope with resources that are only ever useful for a particularly specialized pipeline.
* Pipeline-local resources are natural fit for pipeline templates which use `((vars))` to configure their resources. For example, a branch pipeline template would have a pipeline-local resource definition for its branch.

Concourse will prevent 'resource shadowing' - any pipeline which defines a local resource matching the name of a project-wide resource should error upon configuration.

## Project-level tasks

Projects define tasks under the `tasks/` directory.

Tasks will automatically propagate to every build plan executed within the project via the project's config directory artifact.

The `task` step will be modified to load the task config corresponding to the task name from the project artifact by default.

For example, the following build plan...:

```yaml
plan:
- task: foo
```

...will be interpreted as this:

```yaml
plan:
- task: foo
  file: project-name/tasks/foo.yml
```

By automatically propagating task configs and giving meaning to the name (`foo`), pipeline configs get a bit cleaner: users no longer have to manually pass around a resource for their CI config, and tasks no longer have to specify `file:`.

Before:

```yaml
plan:
- get: ci
  passed: [unit]
- get: concourse
  passed: [unit]
- in_parallel:
  - task: fly-linux
    file: ci/tasks/fly-linux.yml
  - task: fly-darwin
    file: ci/tasks/fly-darwin.yml
  - task: fly-windows
    file: ci/tasks/fly-windows.yml
```

After:

```yaml
plan:
- get: concourse
  passed: [unit]
- in_parallel:
  - task: fly-linux
  - task: fly-darwin
  - task: fly-windows
```

Additionally, Concourse will keep track of which version of the project was used for the execution of the build and ensure it uses the same version if the build is re-triggered.

## API relationships

Teams may have many projects. Projects are cheap and in many cases will replace `fly set-pipeline`. Users are already configuring many distinct pipelines within a team, so it's natural to expect the same of projects. By existing within a team, projects can have simple names without conflicting with projects elsewhere on the Concourse cluster.

Projects have many pipelines. Projects provide a namespace for pipelines, preventing pipeline names from colliding with other pipelines within the same team. This allows pipeline names to be short and simple.

Projects have many builds corresponding to executions of the project's own build plan. These should be visible somewhere prominent in the UI. For simple projects which don't use pipelines this is where a user would expect to see their CI output. For larger projects this is where a user would expect to see their pipelines configured, and check for failed builds which may come from things like pipeline config validation errors.

All builds within the project, including builds of jobs within the project, are associated to a specific version of the project. This version determines the configuration to be loaded and provided to the build when executing the build plan. This information could also be visible somewhere in the UI (perhaps an automatic `get` step at the beginning of the build) so that users know what exact version was used for tasks, resources, and the build plan. This may also be useful for build re-triggering.


## API access control

Projects can be created and destroyed by team members, just like pipelines.


# Technical Decisions

## Exposing only the config directory

This is done so that users don't take a shortcut and use the project resource as both the project config *and* the product code throughout the pipeline.

If they were to have the same bits, users could accidentally override their CI scripts with one provided by a `get` step of the same name as the project, potentially bringing task scripts out of sync with the project-provided task configs.

Here's an example - say I have a small single-repo project and I want to keep all my CI config in the same repo. Example: [Booklit](https://github.com/vito/booklit).

If we provided the whole repo to the build plan, I could have the following `project.yml`:

```yaml
name: booklit
plan:
- task: test
```

Assuming the `test` task looks like this:

```yaml
platform: linux

image_resource: # ...

inputs:
- name: booklit

run:
  path: booklit/ci/tasks/scripts/test
```

This would work just fine - the task would receive the `booklit` input via the project resource.

But what does this look like once I start adopting pipelines? Now I'll want to use `passed` constraints, so I need to define a resource that I can reference with `get` steps. My repo is called `booklit`, so I'll just call it `booklit` too, and create `ci/resources/booklit.yml`:

```yaml
type: git
source:
  uri: https://github.com/vito/booklit
  branch: master
```

Now I have a *project* named `booklit` and a *resource* named `booklit`. This is where the weirdness begins, but it's not just a simple naming mistake - we'll get back to this point soon.

So, next I add my pipeline, `ci/pipelines/booklit.yml`:

```yaml
jobs:
- name: test
  plan:
  - get: booklit
    trigger: true
  - task: test

- name: ship
  plan:
  - get: booklit
    passed: [test]
  # ...
```

And now we have a subtle problem. Both of these builds have a `get: booklit`, and that's going to override the project-provided artifact. Structurally, this is fine, since they're both the same repo, but there's one gotcha: these jobs will be using the task config and/or scripts provided by the `get` step, not from the project config. Now it's possible for the job's config to fall out of sync with the tasks that it's running, because the `passed` constraint will at times provide an older version to the `ship` job.

This can cause errors - for example, if a job has been re-configured to have a different set of inputs than what the older task expects, or if a new task config is being used for an outdated task script. (Depending on how we implement project tasks. Maybe the config comes from the artifact at runtime, too.)

Even worse - you could potentially make a mistake and have your project tasks always running against the code from the *project* resource, and not what's actually under test in the pipeline.

So, what's the solution here? Really, it comes down to this: your product code and your CI config are two different things. This becomes clear right from when you had to write an explicit resource for it in order to use `passed` constraints. Applying `passed` constraints to the project itself doesn't really make much sense - the job config had to come from somewhere in the first place to learn about the `passed` constraints. It's a bit of a chicken-egg problem.

Instead, these should be treated as two separate inputs. Your product code is an input alongside your CI code. This also lets you split the CI code out into a separate repo if you want, and use it for every branch of your product, rather than having a copy on each branch and risk having them fall out of sync.

TODO: finish this thought - wrap it up by suggesting that 


# New Implications

## Smoother learning curve

I think this concept can help make Concourse's learning curve more gradual. Right now the gap between tasks and pipelines is quite large.

Before, the learning curve went like this:

1. Tasks/Resources (`fly execute`, `image_resource`)
1. Jobs/Build Plans/Pipelines (`fly set-pipeline`)

Now it can go like this:

1. Resources (`fly set-project`)
1. Build Plans + Resources (`plan:`, `get:`/`put:`)
1. Tasks (`task:`, `fly execute`)
1. Jobs/Pipelines (`set_pipeline` step/`fly set-pipeline`)
