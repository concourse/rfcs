# Summary

This proposal outlines the addition of a Manual step type to the types of steps
which can be added to a job.


# Motivation

As a pipeline developer attempting to use Concourse to automate infrastructural
concerns, I wish to preview what my infrastructure tools plan to execute before
these tools execute said plan. Many infrastructure-as-code tools may produce
plans which are incorrect or harmful. Applying generated plans without manual
review is dangerous and may result in the loss of data or service availability.
Manual review of the plans is required to ensure that infrastructure changes
are executed safely.

For example, in Terraform one runs `terraform plan -out=path/to/potential.plan`,
which prints a human-readable plan to `stdout` and produces the file
`path/to/potential.plan`. A correct workflow requires pausing, reviewing the
human-readable plan, and only after approving the plan, proceeding to run
`terraform apply path/to/potential.plan` in order to actually execute the plan
and make infrastructural changes.


# Proposal

This proposal advocates for the addition of a `manual` step to the existing
variety of steps which may be used in a job (e.g. `get`, `task`, `put`). When
a `manual` step is added to a job, the Concourse API should expose two options:
an "approve" option, and a "reject" option. If a user should select the
"approve" option, then the manual step should be marked as successful and the
build should proceed normally. If a user should select the "reject" option,
then the manual step should be marked as failed, and the build should respond
accordingly, as if any other step in the build had failed.


## Supervisors

It is occasionally desirable for the authority to approve `manual` steps to be
restricted to a limited subset of users. An RBAC role, `supervisor`, should be
added to Concourse teams. A user with the `supervisor` role has the power to
approve manual steps. Team admins, by virtue of having the power to alter
pipelines and could remove the manual step if they so desired, also have the
power to approve manual steps.


## Pipeline YAML

The following represents a proposed schema for adding `manual` steps to jobs in
pipelines:

```yaml
manual:
  name: required string
  timeout: required duration
  message: optional string
  supervisor_required: optional bool
```

All manual steps must have a _name_, and all manual approval names must be
unique within the context of a single pipeline. This restriction is in place to
ensure that a submitted approval or rejection is directed to the appropriate
`manual` step, in case multiple `manual` steps are open simultaneously as
distributed by an `in_parallel` step, or across multiple jobs within the
pipeline. Attempting to run `fly set-pipeline` on a pipeline where multiple
`manual` steps exist with the same name should result in a validation error
rejecting the proposed pipeline.

Although the _timeout_ step modifier is usually an optional addition to steps,
each manual step must define a timeout step modifier. Manual steps are special
in this requirement (compared to other steps) in order to ensure that forgotten
jobs with open manual approvals, including their task containers and volumes,
are garbage-collected.

All manual steps must have a _message_, which is the prompt asked by the UI
during the `manual` step asking for approval. The default message is:
`Do you permit the build to proceed?`

Manual steps may or may not require the approval of a user with the
`supervisor` role. By default, the `supervisor_required` value is set to
`false`. If the `supervisor_required` value is set to false, then any member of
the pipeline's team has permission to approve or reject the `manual` step. If
the `supervisor_required` value is set to true, then only team members with the
`supervisor` (or, by extension, `admin`) role may approve or reject this
`manual` step.


## Cluster configuration

An additional configuration flag should be added to `web` nodes,
`CONCOURSE_MAX_MANUAL_APPROVAL_DURATION`, which defines the longest permissible
duration which may be defined in the `timeout` field of any `manual` step. If
this configuration flag is specified, then any pipeline which attempts to set a
`timeout` of longer than this maximum should fail validation during
`fly set-pipeline`, with `fly` explaining which `manual` step violates the
restriction and printing what the maximum `timeout` is configured to (a cluster
admin may forbid the use of `manual` steps in this way by setting
`CONCOURSE_MAX_MANUAL_APPROVAL_DURATION` to `0s`, if so, `fly` should recognize
this pattern and exceptionally print that `manual` steps are forbidden on this
cluster).


## UI

### Web

The Concourse Web UI should provide two buttons: an "approve" button, and a
"reject" button. If the "approve" button is pressed, then a request should be
submitted to the API approving the `manual` step. If the "reject" button is
pressed, then a request should be submitted to the API rejecting the `manual`
step.


### `fly`

The `fly` tool should expose a new command: `manual-oversight` (alias: `mo`)
which can submit approvals and rejections. Its command-line options should be
as follows:

* `--pipeline` (required, short: `-p`): in which pipeline the manual step can
  be found
* `--step` (required, short: `-s`): the name of the `manual` step to approve or
  reject
* '--approve` (optional, short: `-a`, mutually exclusive with `--reject`):
  submit an approval for this `manual` step
* `--reject` (optional, short: `-r`, mutually exclusive with `--approve`):
  submit a rejection for this `manual` step

If both the `--approve` and `--reject` options are specified, then `fly` should
exit with an error.

If neither the `--approve` nor the `--reject` options are specified, then `fly`
should prompt the user at the console at run-time for the decision whether to
approve or reject the `manual` step.


# Open Questions

## API Workflow

The process of submiting approvals should be exposed to the end user through the
Concourse API, i.e. by submitting a `POST` request to:
`http://example.com/api/v1/teams/TEAM_NAME/pipelines/PIPELINE_NAME/manual_oversight/MANUAL_STEP_NAME/decide`
This will make it easier to build more advanced integrations with manual
approvals, e.g. building chat bots which permit users to approve or reject
manual steps through the chat bot instead of through `fly` or the web UI.
However, the authentication flow for such an OAuth-enabled workflow still seems
to be an open question (or at least poorly documented).


# Answered Questions

None as of yet.


# New Implications

## "Expired" build status

It would be preferable, if a build with a manual step were to timeout on that
manual step, for the build status to be a new "expired" status rather than
"failed". By making this distinction, the user can more easily distinguish
between a step whose approval was not obtained, and a step which was actively
rejected. There may be downstream effects for the user and/or integrations
depending on the difference between these two statuses. For example, a user's
organizational policy may permit a user to reattempt changes which were not
approved in time, but such reattempts may be forbidden if a supervisor were
to make an active rejection.

This implies that the `try` and `attempts` step modifiers should be reworked -
instead of implying that all failures are the same, they should learn the
difference between various build statuses. For example, a user may wish for a
`manual` step to be be attempted multiple times, through the `attempts` step
modifier, but only if the `manual` step resulted in the `expired` build status
and not in the `failed` build status. This would permit a configuration format
similar to the following:

```yaml
do:
- put: notify-supervisor
- manual:
    name: supervisor-oversight
    supervisor_required: true
- put: deploy
attempts:
  number: 3
  repeat_on_statuses:
  - expired
```

wherein approval is sought multiple times (including asking for approval with
the `put: notify-supervisor` step), but only if the `manual` step expired, and
not if the `put: deploy` step failed.
