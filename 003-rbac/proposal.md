# Summary

This proposal outlines the beginnings of support for Role Based Access Control
(or RBAC for short).


# Motivation

A number of users have been asking for ways to restrict the permissions given
to various users in the system, so this will be an attempt to outline what
these restrictions might look like, and how they might work.

There are various GitHub issues around this whole problem space, namely:

* [#1317](https://github.com/concourse/concourse/issues/1317) (the 'organizing'
  issue for RBAC)
* [#769](https://github.com/concourse/concourse/issues/769) and
  [#970](https://github.com/concourse/concourse/issues/970) for exposing
  read-only access for a pipeline to another team

The scope of RBAC is quite large and there are likely many different
configurations that users will want. This proposal is just a start.


# Proposal

This proposal introduces the concept of a **role**. A role is a name associated
to its authentication requirements.

Each team will support the following roles:

* `owner`: Has read/write access to the team's resources, and can make admin
  changes to the team itself (`set-team`, `rename-team`, etc). This is
  effectively the permissions granted to all team-members prior to this
  proposal, and all teams will have their existing configuration migrated to
  this role upon upgrading.

* `member`: Has read/write access to the team's resources.

* `viewer`: Has read-only access to the team's resources.

For now, these roles are hard-coded into Concourse and are completely static.
Dynamic role management is out-of-scope of this proposal.


## How are roles configured?

Currently, each team has a single authentication configuration, configured via
`fly set-team`.

With this proposal, `fly set-team` will continue to function as-is with the
difference that it will now be configuring the `owner` role. This is to remain
backwards-compatible for users that do not need RBAC functionality.

To configure other roles, a configuration file must be used and passed as the
`--config` (or `-c`) flag instead. This file contains the configuration for all
roles.

For example:

```sh
fly -t my-target set-team -n my-team -c config.yml
```

...with a `config.yml` as follows:

```yaml
roles: 
  owner:
    local: 
      users: ["some-admin"]
  member:
    github: 
      users: ["my-github-login"]
      teams: ["my-org:my-github-team"]
    cf: 
      users: ["myusername"]
      spaces: ["myorg:myspace"]
  viewer:
    local:
      users: ["read-only-user"]
```


## How do roles get persisted in the database?

The configuration for a team's authentication is currently stored in the
database as a JSON object containing the Dex users and groups that are
permitted access. For example, a team permitting the 'Developers' team in the
`concourse` GitHub organization and `pivotal-jwinters` GitHub user would look
something like this:

```json
{
  "groups": ["github:concourse:Developers"],
  "users": ["github:pivotal-jwinters"]
}
```

With roles, this configuration simply becomes nested under each role.

For example, a configuration that preserves the above configuration for the
`owner` role but has a `viewer` role allowing anyone in the `concourse` GitHub
organization would look like this:

```json
{
  "viewer": {
    "groups": ["github:concourse"],
    "users": []
  },
  "owner": {
    "groups": ["github:concourse"],
    "users": ["github:pivotal-jwinters"]
  }
}
```

The `up` migration will move the existing configuration under a `"owner"` key
for backwards-compatibility. To configure other roles or modify the owner role,
this can be changed as normal by using `fly set-team` with a config file as
described above.


## How are roles handled by the API?

Currently, when a user logs in Concourse uses each team's auth config to
determine whether the user has access to each team. These teams are then stored
as a claim in the user's token, looking something like this:

```json
{
  "teams": ["team1", "team2"]
}
```

With this proposal, this approach changes slightly to instead check against the
auth config of each team's *roles*. Any roles matching the user are then stored
in the token associated to the team in a map, like so:

```json
{
  "teams": {
    "team1": ["owner"],
    "team2": ["member", "viewer"]
  }
}
```

Concourse's API endpoints will each be modified to require one of the three
roles for the team.

The proposed endpoint-to-role mapping is as follows:

```go
var requiredRoles = map[string]string{
  atc.SaveConfig:                    "member",
  atc.GetConfig:                     "viewer",
  atc.GetBuild:                      "viewer",
  atc.GetBuildPlan:                  "viewer",
  atc.CreateBuild:                   "member",
  atc.ListBuilds:                    "viewer",
  atc.BuildEvents:                   "viewer",
  atc.BuildResources:                "viewer",
  atc.AbortBuild:                    "member",
  atc.GetBuildPreparation:           "viewer",
  atc.GetJob:                        "viewer",
  atc.CreateJobBuild:                "member",
  atc.ListAllJobs:                   "viewer",
  atc.ListJobs:                      "viewer",
  atc.ListJobBuilds:                 "viewer",
  atc.ListJobInputs:                 "viewer",
  atc.GetJobBuild:                   "viewer",
  atc.PauseJob:                      "member",
  atc.UnpauseJob:                    "member",
  atc.GetVersionsDB:                 "viewer",
  atc.JobBadge:                      "viewer",
  atc.MainJobBadge:                  "viewer",
  atc.ClearTaskCache:                "member",
  atc.ListAllResources:              "viewer",
  atc.ListResources:                 "viewer",
  atc.ListResourceTypes:             "viewer",
  atc.GetResource:                   "viewer",
  atc.PauseResource:                 "member",
  atc.UnpauseResource:               "member",
  atc.UnpinResource:                 "member",
  atc.CheckResource:                 "member",
  atc.CheckResourceWebHook:          "member",
  atc.CheckResourceType:             "member",
  atc.ListResourceVersions:          "viewer",
  atc.GetResourceVersion:            "viewer",
  atc.EnableResourceVersion:         "member",
  atc.DisableResourceVersion:        "member",
  atc.PinResourceVersion:            "member",
  atc.ListBuildsWithVersionAsInput:  "viewer",
  atc.ListBuildsWithVersionAsOutput: "viewer",
  atc.GetResourceCausality:          "viewer",
  atc.ListAllPipelines:              "viewer",
  atc.ListPipelines:                 "viewer",
  atc.GetPipeline:                   "viewer",
  atc.DeletePipeline:                "member",
  atc.OrderPipelines:                "member",
  atc.PausePipeline:                 "member",
  atc.UnpausePipeline:               "member",
  atc.ExposePipeline:                "member",
  atc.HidePipeline:                  "member",
  atc.RenamePipeline:                "member",
  atc.ListPipelineBuilds:            "viewer",
  atc.CreatePipelineBuild:           "member",
  atc.PipelineBadge:                 "viewer",
  atc.RegisterWorker:                "member",
  atc.LandWorker:                    "member",
  atc.RetireWorker:                  "member",
  atc.PruneWorker:                   "member",
  atc.HeartbeatWorker:               "member",
  atc.ListWorkers:                   "viewer",
  atc.DeleteWorker:                  "member",
  atc.SetLogLevel:                   "member",
  atc.GetLogLevel:                   "viewer",
  atc.DownloadCLI:                   "viewer",
  atc.GetInfo:                       "viewer",
  atc.GetInfoCreds:                  "viewer",
  atc.ListContainers:                "viewer",
  atc.GetContainer:                  "viewer",
  atc.HijackContainer:               "member",
  atc.ListDestroyingContainers:      "viewer",
  atc.ReportWorkerContainers:        "member",
  atc.ListVolumes:                   "viewer",
  atc.ListDestroyingVolumes:         "viewer",
  atc.ReportWorkerVolumes:           "member",
  atc.ListTeams:                     "viewer",
  atc.SetTeam:                       "owner",
  atc.RenameTeam:                    "owner",
  atc.DestroyTeam:                   "owner",
  atc.ListTeamBuilds:                "viewer",
  atc.SendInputToBuildPlan:          "member",
  atc.ReadOutputFromBuildPlan:       "member",
}
```
