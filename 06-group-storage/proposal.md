# Summary

In Concourse, when a user logs in, we gather all their team memberships and encode them into their access token. This comes with a number of benefits, however the main drawback is that if they get assigned to a new team, they need to log out and log back in for this to have an effect.

So the goal of this proposal is to have a discussion about the benefits and drawbacks of encoding these teams in the token vs doing a lookup in the database on each API request.


## Encode them in the token (current solution)

The payload of a Concourse token until very recently looked like this:

```
{
  "user_id": 1234567890,
  "teams": ["main", "some-team", "some-other-team"]
}
```

Now that we've introduced `roles` it looks more like this:

```
{
  "user_id": 1234567890,
  "teams": {
    "main": ["owner"],
    "some-team": ["viewer"],
    "some-other-team": ["member"]
  }
}
```

On each API request we verify the token and then extract the teams and roles, and then make sure you're allowed to perform the given action.


### Advantages

- It's efficient. You don't need to perform a db lookup on every request.

- It's a simpler data model. No need to store/manage/migrate this information in a database.


### Disadvantages

- It's eventually consistent. Changes made to the user's memberships won't take effect until they log out and log back in.

- It's operationally complex. We need to issue our own Concourse tokens, which means we need a skymarshal component to take a `dex` token and convert it to a Concourse token.


## Store them in the database

### Advantages 

- It's immediately consistent. You won't have to log out and log in again if you're added to a new concourse team. (This doesn't hold for upstream group memberships)

- It's operationally less complex. We could probably get rid of the skymarshal component altogether and rely directly on `dex` tokens.


### Disadvantages

- It's slower. Every api request requires a database lookup.

- It's a more complex data model. More things to manage in the database.

- We still don't have full control over upstream groups. This means that if you're added to a new team in github, for example, you would still need to log out and log back in for us to pick up those changes.
