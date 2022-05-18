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
that supports the JWT standard.

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

### Potential Workflow in Hashicorp Vault context

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
    auth_path: "/auth/jwt"
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

or (forgive my golang pseudocode, rusty and untested):

```golang
// not dealing with errors or worrying about syntax, this is pseudocode
// but please still make fun of my pseudocode, maybe I'll learn something
func (ap *VarSources.Config.auth_path) FmtAuthPath() nil {
  ap := strings.Trim(ap, "/")
  ap := strings.TrimPrefix(ap, "auth/")
  ap := strings.TrimSuffix(ap, "/login")
  ap := path.Clean(ap)
}
type LoginData struct {
  Role string `json:"role"`
  JWT  string `json:"jwt"`
}
type LoginResp struct {
  auth struct {
    ClientToken string   `json:"client_token"`
    Accessor    string   `json:"accessor"`
    Policies    []string `json:"policies"`
    Lease       int      `json:"lease_duration"`
    Renewable   bool     `json:"renewable"`
  } `json:"auth"`
}
func (c *VarSources.Config) VaultAuth() (token string, accessor string, err error) {
  // I'm sure the vault package implements the auth method much better than this
  client := &http.Client{}
  uri := fmt.Sprintf("%s/v1/auth/%s/login", c.url, c.auth_path.FmtAuthPath())
  jsonData := LoginData{
    Role: c.auth_role,
    JWT:  os.Getenv("CONCOURSE_CI_JWT")
  }
  var buf bytes.Buffer
  err := json.NewEncoder(&buf).Encode(jsonData)
  req, err := http.NewRequest("POST", uri, &buf)
  req.Header.Add("X-Vault-Namespace", c.namespace)
  resp, err := client.Do(req)
  defer resp.Body.Close()
  var dec LoginResp
  dec = json.NewDecoder(&buf).Decode(resp)
  return dec.auth.ClientToken, dec.auth.Accessor, nil
}
```

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
