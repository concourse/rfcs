# Summary

Static Configuration of Concourse Instances

# Motivation

Running Concourse at scale, for a long period of time, poses a number of
problems that make operability difficult.

* In large organizations, it is not feasible for a team to set up Concourse by
  themselves. If Concourse adoption is being driven by the engineering team
  itself (i.e. bottom-up adoption), then the team is often challenged to
  orchestrate the acquisition of servers (for `web` and for workers), storage, a
  PostgreSQL database, networking, and credential access. If Concourse adoption
  is being driven by engineering leadership (i.e. top-bottom adoption), then
  leadership is challenged to orchestrate the synchronization of configuration
  across a wide variety of Concourse instances that have been set up for
  engineering teams.
* For any sufficiently long-lived Concourse instance, database usage grows over
  time, and the underlying PostgreSQL database grows vertically to match,
  increasing the costs for end teams. Reducing reliance upon the underlying
  PostgreSQL database has a direct, positive effect on instance performance and
  costs incurred by Concourse administrators.
* Perusing the current configuration for a Concourse instance, when so much of
  the current configuration lives in the underlying PostgreSQL database, is
  undocumented, difficult and error prone (if automation is attempted), and
  changes are difficult to audit.
* Configuring a `web` node is currently done either through passing flags at
  runtime, or by setting environment variables prefixed with `CONCOURSE_`, as
  [documented](https://concourse-ci.org/concourse-cli.html). While this means
  of configuration is sufficient for providing one-line strings, e.g. setting a
  connection string for a PostgreSQL database, it precludes the addition of more
  complex multi-line configuration.

# Proposal

## Replacing Pre-Existing Command-Line Flags

Permit running a `web` instance with a
`--static-instance-configuration=/path/to/config.yaml` flag. If the
`--static-instance-configuration` flag is provided, then no other command-line
flags will be permitted; attempting to provide other flags (excepting the
`--validate` flag, see below) will result in `concourse web` exiting with an
error. Sending `SIGHUP` to the `web` process will ask the `web` process to
reload this file, without restarting the process.

The contents of this configuration YAML file will permit listing any of the
existing options under `.options`. Sensitive strings may be substituted with
`((FOO))` syntax, indicating that the string to be used is found in the
environment variable `FOO`. For example (only a small subset is included,
for brevity's sake):

```yaml
options:
  external_url: https://concourse.example.com
  enable_lets_encrypt: true
  postgres_user: concourse
  postgres_password: ((CONCOURSE_POSTGRES_PASSWORD))
```

## Replacing `fly set-team`

Doing so permits the configuration file to include additional configuration that
was not possible with the previous command-line flags approach. For example, a
Concourse instance can be configured on bootup to include all of the relevant
teams, and their roles. For example, to replicate the [documented
example](https://concourse-ci.org/managing-teams.html#setting-roles) of setting
user roles, in place of `fly set-team --config`:

```yaml
options:
  # ... omitted for the sake of brevity ...
teams:
- name: foo
  roles:
  - name: owner
    github:
      users: ["admin"]
  - name: member
    github:
      teams: ["org:team"]
  - name: viewer
    github:
      orgs: ["org"]
    local:
      users: ["visitor"]
```

If the `.teams` section of the config file is included, then attempts to call
`fly set-team`, `fly destroy-team`, or `fly rename-team` on this instance will
result in an error.

## Projects

The Projects RFC
[proposes](https://github.com/vito/rfcs/blob/projects/032-projects/proposal.md#proposal)
to use the following call structure to `fly` in order to create a project:

```bash
fly set-project \
  --project booklit \
  --type git \
  --source uri=https://github.com/vito/booklit \
  --source branch=projects-mock \
  --path ci
```

This RFC proposes replacing this call structure by adding the following section
to the configuration YAML:

```yaml
options:
  # ... omitted for the sake of brevity ...
teams:
  # ... omitted for the sake of brevity ...
projects:
- name: booklit
  type: git
  source:
    uri: https://github.com/vito/booklit
    branch: projects-mock
  path: ci
```

This permits bootstrapping a Concourse instance to fetch and setup all pipelines
upon initial bootup. If `.projects` is specified in the configuration YAML, then
any call to `fly set-project` will return an error, and any call to `fly
set-pipeline` will return an error, asking the user to migrate to projects.
Likewise, calling `fly destroy-pipeline` or `fly rename-pipeline` will result in
similar errors.

## The `--validate` flag

If the Concourse binary is run with both the `--static-instance-configuration`
and the `--validate` flag, then the Concourse binary will attempt to validate
the configuration YAML provided and exit, returning an exit code of zero if the
configuration is valid and non-zero if the configuration is not valid.

## Workers

The ability to pass a `--static-instance-configuration` flag when running
`concourse worker` should also be possible, with the current options for
`concourse worker` being made available under `.options`.

# Open Questions

## Nested Options

Now that configuration is handled in a YAML file, the options can now be nested.
For example, one way of rewriting the original configuration could be:

```yaml
options:
  external_url: https://concourse.example.com
  lets_encrypt:
    enable: true
  postgres:
    user: concourse
    password: ((CONCOURSE_POSTGRES_PASSWORD))
```

This improves legibility of the YAML configuration compared to putting all
options in a flat level underneath `.options`, at the expense of making it more
difficult for Concourse administrators to migrate their start scripts/manifests.

# Answered Questions

TBD

# New Implications

## Review Workflows for Administrators

The `--validate` flag will make it possible to set up review workflows for
Concourse configurations kept in version control, adding confidence that
proposed changes will succeed.

## YAML Templating

As configuration becomes increasingly complex, the use of YAML templating
systems by operators has become more popular. By using a YAML templating system,
Concourse administrators can create a template for teams they serve, making it
more simple for a Concourse administrator to "spin-up a Concourse" for their
teams, while ensuring that any changes are evenly applied to all of their
customer teams by changing only the template.

Indeed, an enterprise team that owns Concourse could set up a Concourse instance
for themselves that continuously spawns Concourse instances for the rest of the
teams in their organization, by determining which teams exist in their
organization, generating configuration per team, spinning up cloud
infrastructure for that team, and then deploying Concourse with the generated
configuration onto that newly spun-up infrastructure.
