# Summary

This proposal is meant to start a discussion about Role Based Access Control (or RBAC for short). A number of users have been asking for ways to restrict the permissions given to various users in the system, so this will be an attempt to outline what these restrictions might look like, and how they might work.

One of the main drawbacks of implementing RBAC is that there can be performance implications. These typically come about when we need to make expensive queries to determine wether or not a user can perform the intended action. This isn't a good enough reason not to implement such checks, but we should be mindful of the potential consequences.

We'll try to answer a few questions about how this might work. Specifically:

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

Where `/tmp/team-config` will look something like:


### Team roles config

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

Now that we have roles we're goign to map the team to a list of roles as follows:


### Roles stored in token
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

