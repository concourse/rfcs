# var steps + local var sources

This proposal introduces two new step types - `load_var` and `get_var` - along
with a new mechanism for builds to use a "local var source" at runtime.

* The `load_var` step can be used to read a value from a file at runtime and
  use it as a local var in subsequent steps.

* The `get_var` step can be used to fetch a var from a var source and trigger a
  new build when its value changes.

Both steps save the var in the build's "local var source", accessed through the
special var source name `.` - e.g. `((.:some-var))`.

## Motivation

1. The `load_var` step introduces a general mechanism for using a file's
   contents to parameterize later steps. As a result, resource (proto)type
   authors will no longer have to implement `*_file` forms of any of their
   parameters.

1. The `get_var` step introduces more flexible way to trigger and parameterize
   jobs without having to use a resource.

   * A `vault` var source type could be used to trigger a job when a
     credential's value changes, in addition to its normal use for var syntax.

   * A `time` var source type could be used to trigger jobs on independent
     intervals.

   By invoking the var source type with metadata about the job, the var source
   type can base its behavior on the job in question:

   * A `vault` var source can use the team and pipeline name to look for the var
     under scoped paths.

   * A `time` var source could use a hash of the job ID to produce a unique
     interval for each job.

   With the `time` var source producing a unique interval for each job, this
   will eliminate the "stampeding herd" problem caused by having many jobs
   downstream of a single `time` resource.

   This would in turn allow us to un-feature-flag the long-running ["global
   resource history" experiment][global-resources-issue], which allows
   Concourse to optimize equivalent resource definitions into a single history,
   requiring only one `check` interval to keep everything up to date, and
   lowering database disk usage.

## Proposal

This proposal introduces a new "local var source", accessed through the special
name `.`. This local var source contains vars that were set in a temporary
local scope.

Each build will have a single local scope which steps can use for setting and
getting var values.

This proposal introduces two new steps which set vars in the local scope:
`load_var` and `get_var`.

### `load_var`: loading a var's value from a file at runtime

The following example uses a task to generate a branch name and then
parameterizes a `git` resource with it in order to push to the branch:

```yaml
plan:
- task: generate-branch-name
  outputs: [branch-name]
- load_var: branch-name
  file: branch-name/name
- put: booklit
  params:
    branch: ((.:branch-name))
    base: master
```

In this case, the `name` file will have contained a simple string value.

With secret redaction enabled, the value for `((.:branch-name))` will be
redacted by default.

In this case the value isn't sensitive, and it's probably more helpful to allow
it to be printed. To disable redaction for the var's value, configure
`sensitive: false`:

```yaml
load_var: branch-name
file: branch-name/name
sensitive: false
```

If a filename ending in `.yml`, `.yaml`, or `.json` is referenced, its value
will automatically be parsed and used as fields for the var. Let's tweak the
example to demonstrate:

```yaml
plan:
- task: generate-branch-name
  outputs: [branch-info]
- load_var: branch
  file: branch-info/info.json # {"name":"foo"}
- put: booklit
  params:
    branch: ((.:branch.name))
    base: master
```

In some cases you might not want the value to be parsed as fields for the var.
For example, GCP credentials are JSON format but almost always handed around
verbatim. The automatic parsing can be disabled by explicitly setting `format:
raw`:

```yaml
load_var: gcp-key
file: super-secret/gcp.json
format: raw
```

Similarly, you may have a file with no file extension but still want to parse
it as YAML or JSON. This can be done by explicitly configuring the format:

```yaml
load_var: branch
file: branch-info/info
format: json
```

### `get_var`: triggering on changes from a var source

The following example uses a `time` var source type to periodically trigger a
job:

```yaml
var_sources:
- name: test-interval
  type: time
  config:
    interval: 10m

jobs:
- name: trigger-over-time
  plan:
  # trigger on changes to `((test-interval:time))`
  - get_var: time
    source: test-interval
    trigger: true
```

The following example uses a `vault` var source type to trigger a job whenever
a credential rotates:

```yaml
var_sources:
- name: my-vault
  type: vault
  config:
    url: https://vault.example.com
    ca_cert: # ...
    client_cert: # ...
    client_key: # ...

jobs:
- name: trigger-on-credential-change
  plan:
  # trigger on changes to ((my-vault:cert))
  - get_var: cert
    source: my-vault
    trigger: true
  - put: deployment
    params:
      ca_cert: ((.:cert))

  # trigger on changes to ((my-vault:cert/foo/bar))
  - get_var: cert/foo/bar
    source: my-vault
    trigger: true
  - put: deployment
    params:
      ca_cert: ((.:cert/foo/bar))
```

Build scheduling invokes the var source with a `get` request against an object,
interpreting the response object as the var's values. If the value is different
from the last value used, a new build is triggered. This comparison can be
based on a hash so we don't have to store sensitive credential values.

A `time` var source's input object might look something like this:

```json
{
  "var": "time",
  "interval": "10m",
  "team": "some-team",
  "pipeline": "some-pipeline",
  "job": "some-job"
}
```

Note the addition of `team`, `pipeline`, and `job` - this will be automated by
Concourse. (TODO: The format and contents of these values is something we
should probably put more thought into; we may want it to match the
[notifications RFC][notifications-rfc].)

And the response might look something like this:

```json
{
  "iso8601": "2020-01-18T23:09:00-05:00",
  "unix": 1579406940
}
```

This response would then be loaded into the build's local var source, available
as `((.:time))`, and with its fields read as e.g. `((.:time.iso8601))` or
`((.:time.unix))`.

## Open Questions

* n/a

## Answered Questions

* n/a

## New Implications

1. A separate RFC could be written so that `get` steps can also provide a local
   var containing the object returned by the resource. This could be used
   instead of writing values to files.

[resources-rfc]: https://github.com/vito/rfcs/blob/resource-prototypes/038-resource-prototypes/proposal.md
[global-resources-issue]: https://github.com/concourse/concourse/issues/2386
[notifications-rfc]: https://github.com/concourse/rfcs/pull/28
