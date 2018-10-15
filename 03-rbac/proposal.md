# Summary

Introduces a proposal for RBAC. Specifically:

- What roles do we need?
- How would fly set-team work?
- How do roles get persisted in the database?
- How are roles handled by the API?


# Proposal

## What roles do we need?

`admin` - Granted if you are part of an admin team. Currently only the `main` team is an admin team.

`owner` - Granted during `fly set-team`. Has read/write access to all the team's resources, as well as admin priviledges for the team itself (`set-team`, `rename-team`, etc).

`member` - Granted during `fly set-team`. Has read/write access to all the team's resources.

`viewer` - Granted during `fly set-team`. Has readonly access to all the team's resources.


## How would fly set-team work?

Fly set-team should work the same as before:
```bash
fly -t mytarget set-team -n myteam --allow-all-users
fly -t mytarget set-team -n myotherteam --github-user pivotal-jwinters --github-team myorg:myteam
```

This will assign the `owner` role to the specified users and groups. If you want more control over which roles get assigned, you can provide a config file with the mappings.

```bash
fly -t mytarget set-team -n myteam -c /tmp/team-config
```

Where `/tmp/team-config` might look something like:

### Option 1 - roles array

```yaml
roles: 
- name: owner
  local: 
    users: ["some-admin"]
- name: member
  github: 
    users: ["my-github-login"]
    teams: ["my-org:my-github-team"]
  cf: 
    users: ["myusername"]
    spaces: ["myorg:myspace"]
- name: viewer
  allow_all_users: true
```

or this:

### Option 2 - roles map

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
    allow_all_users: true
```

Or something else?


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


### Option 1 - team:role tuples
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
	...
}
```

This is fine. The API will just cycle through the roles until it finds the one it cares about. 

### Option 2 - Something a bit more structured

Maybe we want something a bit more structure to make changes easier down the road.

```
{
	"is_admin": true,
	"teams": [
	  {"name": "team1", "roles": ["viewer"]},
	  {"name": "team2", "roles": ["viewer", "member"]}
	],
	...
}
```

The main disadvantage here is that it starts to bloat the size of our token. 

Or a slimmed down alternative:

```
{
	"is_admin": true,
	"teams": {
	  "team1": ["viewer"],
	  "team2": ["viewer", "member"]
	},
	...
}
```

