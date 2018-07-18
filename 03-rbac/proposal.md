# Summary

Introduces a proposal for RBAC. Specifically:

- What roles do we need?
- How would fly set-team work?
- How do roles get persisted in the database?
- How are roles handled by the API?


# Proposal

## What roles do we need?

`global_admin` or `admin` - Granted if you are part of an admin team. Currently only the `main` team is an admin team.

`team_member` or `member` - Granted during `fly set-team`. Has read/write access to all the team's resources.

`team_viewer` or `viewer` - Granted during `fly set-team`. Has readonly access to all the team's resources.

`public` - Not a role. This applies to pipelines. Public pipelines will behave as they did before.


## How would fly set-team work?

Currently you fly set-team like the following:
```bash
fly -t mytarget set-team -n myteam --allow-all-users
fly -t mytarget set-team -n myotherteam --github-user pivotal-jwinters --github-team myorg:myteam
```

We could allow roles to be set as follows:
```bash
fly -t mytarget set-team -n myteam --role viewer --allow-all-users
fly -t mytarget set-team -n myteam --role member --github-user pivotal-jwinters --github-team myorg:myteam
```

The main drawback here is that we would have to merge `auth` configuration because `viewers` and `members` would be configured in two separate steps.  The current behaviour overwrites the entire auth config each time, so in keeping with the current behaviour I guess we would overwrite the entire config for each role separately. 

For backwards compatibility, the default role would be `member` if you don't pass the `--role` flag.


#### Maybe time for config files?

An alternative approach would be to start introducing config files for `fly set-team(s)`.

```bash
fly -t mytarget set-teams -c /tmp/team-config
```

Where `/tmp/team-config` might look something like:

```yaml
teams: 
  main: 
    admin: true
    roles: 
      member: 
        github:
          teams: ["myorg:myteam"]
          users: ["pivotal-jwinters"]
  myteam: 
    admin: false
    roles: 
      member:
      	local: 
          users: ["myusername"]
      viewer: 
        allow_all_users: true
```

Or maybe `toml` ¯\\_(ツ)_/¯

```toml
[teams.main]
admin = true
  
[teams.main.roles.member.github]
teams = ["myorg:myteam"]
users = ["pivotal-jwinters"]

[teams.myteam]
admin = false
  
[teams.myteam.roles.member.local]
users = ["myusername"]
  
[teams.myteam.roles.viewer]
allow_all_users = true
```

##  How do roles get persisted in the database?

To allow all users, we store team auth like the following:
```json
{
	"groups": [],
	"users": []
}
```

Or, if users need to belong to a specific github team:
```json
{
	"groups": ["github:myorg:myteam"],
	"users": ["github:pivotal-jwinters"]
}
```

We could store `member`, `viewer` info separately:
```json
{
	"viewer": {
		"groups": [],
		"users": []
	},
	"member": {
		"groups": ["github:myorg:myteam"],
		"users": ["github:pivotal-jwinters"]
	}
}
```

The above would allow all users to view, while only those on a specific github team to manage the team's resources.


##  How are roles handled by the API?

When a user logs into Concourse, we encode all their team memberships into their token, as well as wether or not they belong to any admin teams. The two fields we care about look like this:
```
{
	"is_admin": true,
	"teams": ["team1", "team2"],
}
```

The API uses this information to determine wether or not the request is authorized. It won't allow requests on a team's resource if the user isn't a member of that team.

If we wanted to introduce roles we could simply encode them into the token as follows:
```
{
	"is_admin": true,
	"teams": ["team1:viewer", "team2:member"],
}
```
The API could then look at the user's `team_roles` and determine wether or not to permit the given operation. 

Because a user can meet more than one auth criteria configured for a team, they may end up with the same team with multiple roles in their token:
```
{
	"is_admin": true,
	"teams": ["team1:viewer", "team2:viewer", "team2:member"],
}
```

This is fine. The API will just cycle through the roles until it finds the one it cares about. 



