# Summary

We want to look into having a supported and documented Concourse API. The main purpose for this API amongst users will be to utilize it within 
their own applications. To be able to do this, users need access to tokens that last for an indefinite length. Currently, this doesn't exist within Concourse 
as tokens expire within 24hrs. For users that are currently using the undocumented API this means that they have to continually log back in, which is a pain. 

# Proposal

Our thought is that we can add this token via the CLI as a command like `fly generate-token`. This would be an indefinite token that would be able to be revoked at 
any time. The token would grant access to the current read api endpoints. 

```
$ fly -t ci generate-token
token saved to file
```
The token can then be used by an application to send requests to the API.

# Open Questions

- How would these tokens be revoked if they become compromised. Perhaps they're stored in the database and can be removed? 
- Who has access to generate tokens? Do we just grant access to admins? 
- Should we grant access to every read API endpoint or just a selection? 
  - How does a user define the selection when generating the token?
  - Are tokens team scoped?


# Answered Questions


# New Implications

- Should not change any existing workflows, but it should create a path forward for new ones.  
