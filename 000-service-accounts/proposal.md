# Summary

Teams should be able to define a list of service accounts during `fly set-team`
that can be used in automation.


# Motivation

While Concourse is very good at automating things, it's not so good at
automating itself. We've made progress towards it via things like the
[set_pipeline step][set-pipeline-step] and the future [projects RFC][projects].
However, there are still advanced use cases (like pausing pipelines, jobs,
onboarding teams, pinning resources, etc) which are not solved by an RFC.

Many Concourse operators end up creating local users to be used in scripts to
setup Concourse. However, this presents a security risk since these accounts
usually have god powers as the owner of the main team. It's also difficult to
rotate local users' passwords since it requires a restart of the web node.

# Proposal

Multiple service accounts can be created at the team level and be explicitly
granted permissions to perform automation. They are created with an user
supplied token and can be used in `fly login --team foo --token
'user-provided'` to authenticate to the cluster.

The config passed to `fly set-team` can have a section to create service
accounts for that team.  These service accounts will have a human readable
name, a token, and a list of allowed actions that service account can make.

```yaml
roles:
  ...
service_accounts:
- name: team-onboarding
  token: user-provided
  actions:
  - SetTeam
```

The `name` should be unique and human readable as it will be treated as the
username for audit logging and OPA integrations.

The `token` is provided by the user and will use a simple string comparison to
authenticate a `fly login`. It should be redacted in the output of `fly
get-team`. Concourse should not be responsible for generating or rotating this
token.

The `actions` lists all the [actions][action-matrix] that service account would
be allowed.


# Open Questions

- Service accounts encourages the idea of "automating the automation", and can
  be used to make it harder to identify missing features in the core product.
  Is there a better way to make service accounts less desirable to use for the
  average user (and prompting feature requests) but still unblocking power
  users? Maybe service accounts should be restricted from common actions
  (`SetPipeline`, `CreateJobBuild`, `RerunJobBuild`, etc)?


# Answered Questions

# New Implications


[action-matrix]: https://concourse-ci.org/user-roles.html#action-matrix
[set-pipeline-step]: https://github.com/concourse/rfcs/blob/master/031-set-pipeline-step/proposal.md
[projects]: https://github.com/concourse/rfcs/pull/32
