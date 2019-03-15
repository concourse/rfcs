# Summary

Move credential management configuration to the team configuration level, and
allow configuring multiple credential managers for each team.


# Motivation

Configuring a single credential manager is very limiting. Because there is only
one point of authentication configured, each credential manager has to support
some form of multi-tenancy. The current strategy is to encode team and pipeline
names in the paths for the keys that are looked up, but this has many
downsides:

* This makes it impossible to share credentials between teams. Instead the
  credential has to be duplicated under each team's path.

  By moving credential manager config to each team we can instead leverage the
  credential manager's access control to determine how credentials are shared
  across teams (e.g. Vault policies).

* With Vault, this makes it impossible to use any backend except `kv`, because
  all keys live under the same path scheme, and different backends can't be
  mounted under paths managed by other backends. This removes a lot of the
  value of using Vault in the first place.

  By eliminating the path enforcement you can now refer to different secret
  backend mount points.

* Some credential managers, e.g. Azure KeyVault, have very strict requirements
  for key names (`[a-z\-]+`), effectively making scoping conventions impossible
  to enforce.

  By configuring at the team level, each team can point to their own KeyVault
  or configure their own access control.

* It would be nice to be able to leverage a specialized credential manager like
  IAM/STS for some things (like the `s3` resource) and use Vault for everything
  else. Right now you can only configure one credential manager, so this is
  impossible.

  By allowing teams to configure multiple credential managers, all credential
  managers can be tried in order when looking up a given credential.


# Proposal

The first step is to extend the team config file set by `fly set-team --config`
to support configuring credential managers. Something like this:

```yaml
roles: # ...

credential_managers:
- type: iam
  config:
    access_key: blahblah
    secret_key: blahblah
- type: vault
  config:
    url: https://vault.example.com:8200
    ca_cert: |
      -----BEGIN CERTIFICATE-----
      ...
    client_token: blahblahbla
```

Then, any time we're resolving a `((var))` the `web` node would resolve the var
using each configured credential manager, in order. Distinct fields can be
accessed like `((foo.bar))`, and nested credential paths can be accessed like
`((foo/bar/baz))`.

In this case, team's Vault auth config would be associated to a policy which
determines which credentials the team can access. This way shared credentials
can be shared without duplicating the credential, and private credentials can
be kept private.

All credential managers would be modified to remove the automatic team/pipeline
variable name scoping. They would instead be looked up starting from the root
level.


# Open Questions

* Is there a need for globally-configured and team-configured credential
  managers to coexist, or can we switch to entirely team-configured (as is the
  initial goal)?

* Concourse will now be responsible for safely storing access to each and every
  credential manager, which increases risk. Is it enough to mitigate this by
  requiring that database encryption be configured?

* Will anyone miss the automatic credential scoping assumptions? Is there value
  in automatically looking under `/(team)/(pipeline)` for `((foo))`?

* Would a default key prefix make sense? e.g. `/concourse`? (Maybe this is just
  up to the discretion of the credential manager?)

* Supporting IAM/STS token acquisition is one of the motivators for this
  proposal, but I think we need a concrete example implementation to really
  understand if this proposal is a good fit. The above example configures an
  access key and secret manually instead of using EC2 IAM role.

* When and how often do we authenticate with each credential manager? If you're
  using Vault with a periodic token, something will have to continuously renew
  the token.

  Will the `web` node have to maintain long-lived clients for accessing each
  configured credential manager across all teams? Is that going to be a
  scalability concern? Is that going to be a security concern? (Can it be
  avoided?)

  Should it detect situations where this is required?

  Should we just not support periodic tokens?

* Should we work credential caching into this proposal?

* How should credential manager authentication errors be surfaced?

* Is there a need for configuring a path prefix (e.g. the default `/concourse`
  for Vault)? I've left that out for now assuming we can just get rid of it.


# Answered Questions


# New Implications
