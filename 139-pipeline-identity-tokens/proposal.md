* RFC PR: [concourse/rfcs#139](https://github.com/concourse/rfcs/pull/139)

# Summary

Pipelines should receive signed JWTs ([RFC7519](https://datatracker.ietf.org/doc/html/rfc7519)) from Concourse that contain information about them (team, pipeline-name etc.).
They could then send these JWTs to outside services to authenticate using their identity as "Concourse-Pipeline X"


# Motivation
Often pipelines have to interact with outside services to do stuff. For example download sources, upload artifacts or deploy something to the Cloud.
As of now you would need to create static credentials for these outside services and place them into concourse's secrets-management (for example inside vault). Or attach some kind of trust to the concourse-workers (and therefore to ALL pipelines running there).

However having static (long lived) credentials for something that is critical (like a prod account on AWS) is not state of the art for authentication.
And attaching trust to workers means you would need multiple workers with different trust configured, if not all the pipelines share the same "trustworthiness".

It would be much better if code running in a pipeline could somehow prove it's identity to the outside service. The outside service could then be configured to grant permissions to a specific pipeline (or job or task).

Lots if other services already implement something like this. One well known example of this are [Kubernetes's Service Accounts](https://kubernetes.io/docs/concepts/security/service-accounts/#authenticating-credentials). Kubernetes mounts a signed JWT into the pod and the pod can then use this token to authenticate with Kubernetes itself or with any other service that has a trust-relationship with the Kubernetes-Cluster.

## Usage with AWS
For example a Pipeline could use AWS's [AssumeRoleWithWebIdentity API-Call](https://docs.aws.amazon.com/STS/latest/APIReference/API_AssumeRoleWithWebIdentity.html) to authenticate with AWS using it's concourse-token and do stuff in AWS. It is even [directly supported by the AWS CLI](https://docs.aws.amazon.com/cli/latest/reference/sts/assume-role-with-web-identity.html)

1. Create an OIDC-Identity-Provider for your Concourse Server in the AWS Account you would like to use. Like [this](img/AWS-IDP.png).
2. Create an AWS.IAM-Role with the required deployment-permissions and the following trust policy:
```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": "sts:AssumeRoleWithWebIdentity",
            "Principal": {
                "Federated": "<ARN of the Identity Provider of Step 1>"
            },
            "Condition": {
                "StringEquals": {
                    "<ARN of the Identity Provider of Step 1>:sub": [
                        "main/deploy-to-aws"
                    ],
                    "<ARN of the Identity Provider of Step 1>:aud": [
                        "sts.amazonaws.com"
                    ]
                }
            }
        }
    ]
}
```
This trust-policy allows everyone to assume this role via the AssumeRoleWithWebIdentity API-Call, as long as he has a JWT, signed by your Concourse, with the sub-value of "main/deploy-to-aws".

And conveniently Concourse will create exactly such a token and supply it to (and only to) the pipeline "deploy-to-aws" in the "main" team.

When code inside a pipeline performs the AssumeRoleWithWebIdentity API-Call, AWS will check the provided token for expiry, query concourse to obtain the correct signature-verification key and use it to check the JWT's signature. It will then compare the aud and sub claims of the token with the ones specified in the Role's trust policy. If everything checks out, AWS will return temporary AWS-Credentials that the pipeline can then use to perform actions in AWS.

In a concourse pipeline all of this could then look like this:
```yaml
- task: get-image-tag
  image: base-image
  config:
    platform: linux
    run:
    path: bash
    dir: idp-servicebroker
    args:
    - -ceux
    - aws sts assume-role-with-web-identity --d
      --provider-id "<ARN of the Identity Provider of Step 1>" \
      --role-arn "<ARN of the role to be assumed>" \
      --web-identity-token (( idtoken:token ))
    - // do stuff with the new AWS-Permissions
```


## Usage with vault
The feature would also allow pipelines to authenticate with vault. This way a pipeline could directly access vault and use all of it's features and not only the limited stuff that is concourse provides natively.

Vault has support for [authentication via JWT](https://developer.hashicorp.com/vault/docs/auth/jwt).
It works similarly to AWS. You tell Vault an URL to the Issuer of the JWT (your concourse instance) and configure what values you expect in the token (for example the token must be issued to a pipeline of the main team). You can then configure a Vault-ACL and even use claims from the token in the ACL. Your ACL could for example allow access to secrets stored in `/concourse/<team>/<pipeline>` to any holder of such a JWT issued by your concourse.

Detailed usage-instructions for vault can follow if required.

# Proposal
Implementation is split into different phases that stack onto each other. We could implement the first few and expand the implementation step by step.

## Phase 1
- When Concourse boots for the first time it creates a signature key-pair and stores it into the DB. For now we generate a 4096 bit RSA-Key so we can use the RS256 signing method for the tokens. This seems to be the signing method with most support and is used by others for similar purposes ( https://token.actions.githubusercontent.com/.well-known/jwks , https://kubernetes.io/docs/reference/command-line-tools-reference/kube-apiserver/ ). Other key types like EC256 can be added later and would be selectable via the var-source.
- Concourse exposes the public part of the key as a JWKS ([RFC 7517](https://datatracker.ietf.org/doc/html/rfc7517)) under a publicly accessible path (for example: https://myconcourse.example.com/keys)
- Concourse offers a minimal OIDC Discovery Endpoint ([RFC8418](https://datatracker.ietf.org/doc/html/rfc8414)) that basically just points to the JWKS-URL
- There is a built-in var source (see section below) that pipelines can use to get a signed JWT with the following contents:
```json
{
    "iss": "https://myconcourse.example.com",
    "exp": "expiration-time",
    "iat": "time-when-token-was-issued",
    "jti": "nonce",
    "aud": [<configurable via var-source-config>],
    "sub": "team/pipeline-name",
    "team": "team-name",
    "pipeline": "pipeline-name",
    ...<whatever else might be relevant>...
}
```
- That JWT is signed with the signature key created in the beginning
- The jobs/steps of the pipeline use the token to do whatever they like with it
- The sub-claim's value is by default of form `<team>/<pipeline>` (but can be configured, see below)
- Tokens can have an optional aud-claim that is configurable via the var-source (see below)
- Tokens do NOT contain worker-specific information
- If implementable with reasonable effort: The token should contain the job and task/step name

### The IDToken Var-Source
The var-source of type "idtoken" can be used to obtain the tokens described above. It offers a few config-fields to configure the token that is received:

- `subject_scope` string - Specifies what should be included in the sub-claim of the token. The var-source MUST make sure that no component of the sub-claim contains any forward-slashes (`/`) and should escape all slashes by URL-Encoding them.
  - with a value of one of:
    - `team`: `<team>`
    - `pipeline`: `<team>/<pipeline>`
    - `job`: `<team>/<pipeline>/<job>`
    - `step`: `<team>/<pipeline>/<job>/<step_name>`
  - default: `pipeline`

- `audience` []string - The aud-claims to include in the token.
  - default: Empty array

- `expires_in` `time.Duration` - How long the generated token should be valid.
  - default: `1h`
  - Max value accept is `24h`

The output variable of the var-source that contains the token is called `token`. All other variables are reserved for future use.

In the future it would be possible to add a `signature_algorithm` config field that allows the user to choose between RS256 and ES256 as signature-algorithms for his token. (Concourse would need to have one key for each supported algorithm stored).

In the pipeline it would then look like this (all config fields are optional and are shown here for clarity):

```yaml
var_sources:
- name: idtoken
  type: idtoken
  config:
    subject_scope: pipeline
    audience: ["sts.amazonaws.com"]
    expires_in: 1h

jobs:
- name: print-credentials
  plan:
  - task: print
    config:
      platform: linux
      image_resource:
        type: registry-image
        source: {repository: ubuntu}
      params:
        ID_TOKEN: ((idtoken:token))
      run:
        path: bash
        args:
        - -c
        - |
          echo "token: $ID_TOKEN"
          // or
          echo "token: ((idtoken:token))"
          // Send this token as part of an API-Request to an external service
```

## Phase 2
Concourse could periodically rotate the signing key it uses. Default rotation-period will be 7 days. The new key will then also be published in the JWKS and will be used to sign tokens from then on. The previous key MUST also remain published for 24h (which is the absolute maximum lifetime of a token), in case there are still unexpired tokens out there that were signed with it.

The rotation-period should be configurable as an atc-setting. Setting the period to 0 effectively disables automatic key-rotation.

## Phase 3
To make sure tokens are as short-lived as possible we could enable online-verification of tokens. Concourse could offer a Token-Introspection-Endpoint ([RFC7662](https://datatracker.ietf.org/doc/html/rfc7662)) where external services can send tokens to for verification.
That endpoint could reject any token that was issued for a pipeline/job/task that has already finished running.

# Open Questions
(1-7) have already been answered

8. How do pipeline identity tokens work with resources?

# New Implications

This could fundamentally change the way how pipelines interact with external services, making it much more secure.
As JWT-Authentication is a modern standard that is supported by lots of services, it could enable a whole bunch of new usecases.
Use of this feature is entirely optional. Everyone who doesn't need it can completely ignore it.
