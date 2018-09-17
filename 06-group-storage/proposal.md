# Summary

The goal here is to start a discussion about how to store a user's team memberships. So far we've talked about two different options:

- Store them in the token (this is what we do now)
- Store them in the database


## Store them in the token

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
