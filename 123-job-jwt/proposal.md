# Summary

Mint a signed JSON Web Token (JWT) and deliver to the job runtime container.
Additionally, provide a validation endpoint to allow for external or internal
validation of the issued JWT.

## Motivation

Concourse runners lack an identity mechanism that can be used to authenticate to
external systems. This lack means that an initial secret must be provided to
each pipeline in Concourse in a manual manner. Many other CI solutions and cloud
providers solve the "secret zero" problem by provisioning a signed JWT to their
workloads automatically, and providing a path to signature verification in a
manner that the external applications can trust.

Should this be implemented, Concourse jobs will be able to leverage this
automatically generated JWT to be the "secret zero" needed to communicate with
external systems (such as Hashicorp Vault, AWS, and others) and will no longer
require a Concourse administrator (or person configuring a pipeline) to provide
a static credential to Concourse. This alleviates the need for a human to handle
sensitive credentials, increases security by ensuring credentials are not
statically configured and JWTs are ephemeral to each run, and reduces
Concourse's need to properly handle and encrypt data (can be limited to storing
the JWT private signing key).

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

Because we are proposing that the JWT be used externally with products such as
Hashicorp Vault, the JWT would do nothing unless configured to be trusted by the
external system. Thus, the impact to existing pipelines would be minimal.

## Appendixes

### Potential Workflow in Hashicorp Vault context

A Hashicorp Vault operator would have to mount and configure the JWT Auth Method
like so:

```sh
curl -X POST -H "X-Vault-Request: true" -H "X-Vault-Token: $(vault print token)" -d '{"type":"jwt","description":"","config":{"options":null,"default_lease_ttl":"0s","max_lease_ttl":"0s","force_no_cache":false},"local":false,"seal_wrap":false,"external_entropy_access":false,"options":null}' https://127.0.0.1:8200/v1/sys/mounts/concourse
```

```yaml
var_sources:
- name: hvjwt
  type: vault
  config:
    url: https://vault.example.com
    path_prefix: "/concourse"
    lookup_templates: ["/{{.Team}}/{{.Pipeline}}/{{.Secret}}", "/{{.Team}}/{{.Secret}}"]
    shared_path: "/shared/{{.Pipeline}}/{{.Secret}}"
    namespace: "/root"
    auth_path: "/auth/concourse"
    auth_type: "jwt"
    auth_role: "example"

jobs:
- name: use-vars
  plan:
  - task: use-vars
    config:
      platform: linux

      image_resource:
        type: mock
        source: {mirror_self: true}

      run:
        path: sh
        args:
        - -exc
        - |
          test "((hvjwt:some_secret.username))" = "hello-im-a-username"
          test "((hvjwt:"secret with_spaces.etc".some_key))" = "some-value"

```

This would represent a reasonable configuration. If it is determined that
`auth_params` field in the `var_source.config.vault_config` schema should be
re-used, then `auth_params: {role: "example", jwt: "((.CONCOURSE_CI_JWT))"}`
might be a reasonable value, assuming that local environment variables can be
referenced in such a way and we have pre-populated `CONCOURSE_CI_JWT` with the
signed JWT during container pre-flight. However, this format would require more
awareness on the part of the person configuring the pipeline, and the `jwt`
field in the `auth_params` argument would always be the same, so it seems to
make sense to abstract it in this case.

Then, the call to the Vault API for the authentication step might look like the
following:

```sh
#!/usr/bin/env bash
auth_path=$(echo ${auth_path} | sed -E "s|^/*||g" | sed -E "s|/*$||g" |
  sed -E "s|^auth/?||g" | sed -E "s|/?login$||g" | sed -E "s|/+|/|g"
)
resp=$(curl -s -X POST -H "X-Vault-Namespace: ${namespace}" \
  --data @payload.json ${url}/v1/auth/${auth_path}/login
)
export VAULT_TOKEN=$(echo "${resp}" | jq -r ".auth.client_token")
```
