# Summary

Mint a signed JSON Web Token (JWT) and deliver to the job runtime container.
Additionally, provide a validation endpoint to allow for external or internal
validation of the issued JWT.

## Motivation

Bluntly, the native integration with Hashicorp Vault is scoped only to static
secrets and this implementation is inflexible. By leveraging JWT auth at a job
level, the claims can become incredibly granular, and allow for flexible policy
assignment. Additionally, this type of framework would allow authentication to
any number of other frameworks that support the same standard. Then, a more
complete Vault integration can be built on top of this, as could any integration
that supports the JWT standard. Lastly, it removes the need for Concourse to
connect to only one Vault instance, only one Vault namespace, only one Vault
engine, and have Concourse show up as a single entity in Vault audit logs with a
policy that is far from being least privilege.

With a JWT Vault Authentication model, each team could have their pipelines
connect to their own isolated instance or namespace with policy sets tailored to
the needs of the pipeline jobs, which would enable least privliege models. This
would also scale better from the Vault perspective, and instead of Concourse
having to deliver secrets securely from a global persona, a simple trust
relationship can be established to the Concourse JWT validation API endpoint.

Enabling JWT even opens up the ability to allow Concourse pipelines to
authenticate and authorize to arbitrary custom APIs if desired.

## Proposal

Adopt [RFC 7519](https://datatracker.ietf.org/doc/html/rfc7519) and a Go library
that [implements](https://github.com/golang-jwt/jwt) it. Additionally adopt
[RFC 7517](https://datatracker.ietf.org/doc/html/rfc7517) and a Go library that
implements it. (Perhaps [jwx](https://github.com/lestrrat-go/jwx)?)

Create a struct that includes as many unique job metadatum as possible,
including standard claims of `iss`, `sub`, `aud`, `exp`, `jti`, `nbf`, `iat`,
and Concourse standard objects such as `team`, `teamid`, `job`, `jobid`,
`pipeline`, `pipelineid`, `task`, `buildid`, and any other data points that
describe the job and could drive authentication and authorization claims.

Create a public API endpoint that responds to requests where the signed JWT is
part of the payload, likely leveraging JWKS.

## Open Questions

- Where should the claims be aggregated and signing portion be completed? (atc?)
- How should the signed JWT be delivered to the container runtime?
- Where should the validation endpoint live? (web?)
- Where should the signed JWT live? (Environment Variable + File?)
- Should the validation step fail after the container dies regardless of `exp`
claim? (JWT only valid for period equal to job runtime)

## Answered Questions

(This section intentionally left blank)

## New Implications

Since it is a new feature, placed into optional paths, it should not
significantly impact existing workflows. A potential issue may arise if the
pipeline defines resources or executes job tasks that interact with the proposed
locations for the signed JWT in the environment. However, if the placement of
the JWT happens early enough in the lifecycle, this can be mitigated as the
environment variable would simply be overwritten by the user config. A file
location would present a harder collision, but there should be reserved paths
for Concourse artifacts, potentially even mounted to the container as a
read-only filesystem.

Since we are proposing that the JWT be used externally with products such as
Hashicorp Vault, the JWT would do nothing unless configured to be trusted by the
external system. Thus, the impact to existing pipelines would be minimal.
