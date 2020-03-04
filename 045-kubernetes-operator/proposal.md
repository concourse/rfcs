# Summary

By providing a [Kubernetes Operator], Concourse can be made easier to manage,
reducing even more the need for manual intervention, as well as easier to
bootstrap and more secure by default.

ps.: while everything covered here *can* be done (and already is) by those
operating Concourse at large scale (mostly with the help of [BOSH]), it's a lot
of toil that could be automated and made available to anyone using Concourse on
Kubernetes.



# Motivation

The main motivation is **lowering the barrier to, in a declarative manner,
deploying Concourse installations on Kubernetes with the operational best
practices baked into it by default**.

We're currently unable to achieve that given that:

1. despite `web` being the component that is the easiest to run (it stores no
   state), it can be tricky to get it set up properly, specially when you don't
   have the niceties that [BOSH] brings to the table with regards to generating
   and managing certificates & secrets

2. there are certain operational aspects involved with deploying Web nodes that
   are still manual, making automated maintenance of a fleet of Concourse
   installations hard. 


Neither of those are currently tackled by the current [Concourse Helm Chart]:
someone still needs to manually generate those secrets beforehand, and certain
manual operations (e.g., rolling back to an older version) still need to be
manually performed.


### secret generation

Those who leverage the [Concourse BOSH release] might have never even noticed
that Concourse needs certain keys and passwords to be generated, and passed
along through both `web` and `worker` nodes.

That's because under the hood, BOSH took care of doing all of that:


```yaml
instance_groups:
- name: web
  jobs:
  - release: concourse
    name: web
    properties:
      token_signing_key: ((token_signing_key))

variables:
- name: token_signing_key
  type: rsa
```

(from [concourse-bosh-deploymnet/cluster/concourse.yml](https://github.com/concourse/concourse-bosh-deployment/blob/b9ed7d73431486d55f99b010fc861ce3200e26b1/cluster/concourse.yml#L78-L86))

Comparing that to the [Concourse Helm Chart], we can see a stark contrast

```yaml
secrets:
  ## Concourse Session Signing Keys.
  ## Ref: https://concourse-ci.org/install.html#generating-keys
  ##
  sessionSigningKey: |-
    -----BEGIN RSA PRIVATE KEY-----
    MIIEowIBAAKCAQEAwLql/rUIaI+PX7Tl3FWcTee4sQf8/daakALXx955tPwkhqlY
    e4T2V84p/ylFvNWpM4vfcMYKfMY0JLKgAgBvJhCytSkDBhTBoWmN6yE0AB11P9En
    lIZRBWNYqaC2cSge2ZD8qOSnwfFhnQAW8+7pE+ElJAVh7dtdF3A478H50lIigq8I
    ...
```

(from [concourse/concourse-chart's value.yaml](https://github.com/concourse/concourse-chart/blob/43be63a9bafcba1b47f038786d69e0c8f20b1f13/values.yaml#L2025-L2032))

making it easy to tell how bad the chart's approach is (pushes to the cluster
admin to know that those defaults exist and that should be changed with the
proper set of keys).

ps.: we did make this easier by embedding to the `concourse` binary the
`generate-key` subcommand (see [generating keys]), however, that can still be
seen as a manual step.


Important credentials to be generated include:

- database encryption key: https://concourse-ci.org/encryption.html
- [tsa]'s and workers key sets
- session signing key (or the new equivalents once [generic tokens] land)
- local users' passwords


Having support for the generation of such credentials, one can make a deployment
of Concourse secure by default much simpler than if having to ask folks to come
up with their own certs, etc.



### rollbacks

The path to upgrading Concourse is very straightforward: deploy the new image,
and [ATC] will take care of running those "up" migrations for you.

> To upgrade a web node, stop its process and start a new one using the newly
> installed concourse. **Any migrations will be run automatically on start**. If
> web nodes are started in parallel, only one will run the migrations.

(from [restarting & upgrading])

Being so easy and hard to get wrong, one can let continuous deployment tools to
that without any problems.

The other way around though (rollbacks), is not the same: one needs to manually
run `concourse migrate` with not only the right migration ID that it should go
downwards to (the command helps you out with this), but also complete access to
the database.


> if you're stuck in a pinch and need to downgrade from one version of Concourse
> to another, you can **use the concourse migrate command**.

(from [downgrading])


```
Usage:
  concourse [OPTIONS] migrate [migrate-OPTIONS]

[migrate command options]
          --encryption-key=                                          
          --current-db-version                                       
          --migrate-db-to-version=                                   

    PostgreSQL Configuration:
          --postgres-host=                                           
          --postgres-port=                                           
          --postgres-socket=                                         
          --postgres-user=                                           
          --postgres-password=                                       
          --postgres-sslmode=[disable|require|verify-ca|verify-full] 
          --postgres-ca-cert=                                        
          --postgres-client-cert=                                    
          --postgres-client-key=                                     
          --postgres-connect-timeout=                                
          --postgres-database=                                       
```

While we try our best to never need to perform a rollback, that can eventually
be necessary. This is especially true for the Concourse team itself, where we
want to have Concourse being continuously deployed, while making bad deployments
"no-factor" situations.


### other operational aspects that could be automated

Aside from those two cases, there are other (probably less interesting) ones
that could also be made better.


#### db encryption key rotation

To rotate the database encryption key, one needs specify the right set of flags
to `concourse web` so that it can go from the old encryption to the new one.

With a schedule to rotate those keys, we could automate such process and provide
the right flags.

See [rotating the encryption key].


#### maintenance mode

With the introduction of component tracking (see [#4534]), it's now possible to
tweak the intervals at which certain components of Concourse run. 

This means that in the case of *fire* (or some form of maintenance in the
infrastructure), one can, for instance, disable a given set of components (say,
scheduling of builds and checking or resources) altogether, rendering the
installation able to accept inputs from the users, while still sending no load
to the workers (which could even not exist at the moment).

Tying that to the recently added ability of setting "messages for the users"
([#5019]), one could transition a given installation to "maintenance mode" by
switching a flag on the Kubernetes object that represents that installation
(which could be done in a "gitops fashion").


[#4534]: https://github.com/concourse/concourse/issues/4534
[#5019]: https://github.com/concourse/concourse/pull/5019


# Proposal

Have a Concourse [Kubernetes Operator] to deal with the whole deployment of
Concourse on Kubernetes installations.

At a high-level, one should be able to submit to a Kubernetes cluster an
object that represents a Concourse installation:


```yaml
apiVersion: concourse.concourse-ci.org/v1
kind: Concourse
metadata:
  name: hush-house
spec:
  replicas: 2
  image: concourse/concourse-rc:6.0.0-rc.11
  postgresCredentials: pg-secret
  config:
    clusterName: hush-house
    defaultDaysToRetainBuildLogs: 30
    tracing:
      jaeger:
        endpoint: http://simples-collector.jaeger.svc.cluster.local:14268/api/traces
        service: hush-house-web
```

Our controller would then notice the desire to have a cluster setup, and then
act upon that, generating the credentials and saving them as secrets, creating
the necessary constructs to achieve the desired state (e.g., at minimum, a
[Deployment] and a [Service]), and then let the other controllers deal with the
actual deployment.

![](./creation.svg)


With an installation up and running, those admins wanting to perform updates,
would then update that original spec (`kind: Concourse`), which our controller
would react to:


![](./updates.svg)



# Open Questions

### would the Chart still have a place?

I think so.

There's still a not of Kubernetes objects that would need to be generated, to
have this installed, and with the need of somehow templating that, Helm could be
the tool we use for that.


### what about postgresql?

TODO

i. provide credentials, etc to a pg that has already been installed
ii. use something like crossplane


# Answered Questions

> If there were any major concerns that have already (or eventually, through
> the RFC process) reached consensus, it can still help to include them along
> with their resolution, if it's otherwise unclear.
>
> This can be especially useful for RFCs that have taken a long time and there
> were some subtle yet important details to get right.
>
> This may very well be empty if the proposal is simple enough.


# New Implications

> What is the impact of this change, outside of the change itself? How might it
> change peoples' workflows today, good or bad?


[ATC]: https://concourse-ci.org/architecture.html#component-atc
[BOSH]: https://bosh.io/docs/
[Concourse BOSH release]: https://github.com/concourse/concourse-bosh-release/
[Concourse Helm Chart]: https://github.com/concourse/concourse-chart
[generating keys]: https://concourse-ci.org/concourse-generate-key.html
[generic tokens]: https://project.concourse-ci.org/projects/MDc6UHJvamVjdDM1ODI3NzI=
[tsa]: https://concourse-ci.org/architecture.html#component-tsa
[restarting & upgrading]: https://concourse-ci.org/concourse-web.html#restarting-and-upgrading
[downgrading]: https://concourse-ci.org/concourse-web.html#downgrading
[rotating the encryption key]: https://concourse-ci.org/encryption.html#rotating-the-encryption-key
[Kubernetes Operator]: https://kubernetes.io/docs/concepts/extend-kubernetes/operator/
