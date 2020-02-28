# Summary

In a world where Concourse is part of a suite of other applications in
[Kubernetes] which all speak "the kubernetes language" (are somehow setup
through the use of Kubernetes objects), e.g. [kpack], having Concourse also
speaking that language might bring great opportunities for making it easier for
users and operators (as well as tools) to use / manage it.

With the use of [custom resources] we can bring to those "kubernetes speakers"
Concourse constructs that makes it a more native citizen in the "ecosystem".

```
  USER 
        (authorized for `team-1` namespace)

        kubectl --namespace team-1 \
          apply -f <(echo 'kind: Pipeline...')

            => creates pipeline obj


  CONCOURSE

        reconciliates

          applies that pipeline definition to its own
          internal datastore (postgres)

              deals w/ it

          <= gives the feedback through status updates in
             the pipeline obj
```


pps.: prior work https://github.com/jchesterpivotal/knative-build-pipeline-poc



# What this is *not* about

This proposal DOES NOT try encompass:

- deployment of Concourse in a Kubernetes environment: while it's true that we
  could go with the [operator pattern] to get Concourse deployment (where we
  could define a `Concourse` resource with a controller watching for an creating
  `concourse web` pods and a `postgres` database), that's not in the scope of
  this proposal.


- the runtime aspect of Concourse - going from a build plan to steps that at
  some point run as pods in a Kubernetes cluster is in the realm of [rfc: how do
  we best leverage k8s as a runtime] and [k8s-based implementation of atc's
  worker iface] instead.

[operator pattern]: https://kubernetes.io/docs/concepts/extend-kubernetes/operator/
[kpack]: https://github.com/pivotal/kpack
[Kubernetes]: https://kubernetes.io/
[custom resources]: https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/
[k8s-based implementation of atc's worker iface]: https://github.com/concourse/concourse/issues/5209
[rfc: how do we best leverage k8s as a runtime]: https://github.com/concourse/rfcs/pull/22


# Motivation

### kubernetes native integration

Tools that expect to interact with applications that run on Kubernetes through
Kubernetes objects are not able to do so without some form of a translation
layer.



### automatically setting up pipelines

**1. A Concourse installation cannot be automatically set up with a set of
pipelines without some form of manual bootstrap.**

Concourse has always been about being able to continuously doing things, with
the definition of that automation coming right from code.

That means that all of the input to Concourse takes the form of configuration
files that get applied to Concourse through its API.

However, without other forms of automation there's currently no way that a
Concourse installation can have a set of pipelines automatically set on it.

As an example, in our very own [ci.concourse-ci.org] installation, we boostrap
that process by having a pipeline that takes care of setting all of our other
pipelines leveraging the [`set_pipeline`] step (see [`reconfigure.yml`]), not to
mention that given that `set_pipeline` is scoped to the team where the job is
running, one would have to do a `set-pipeline` for a `reconfigure` pipeline for
each team.

[`reconfigure.yml`]: https://github.com/concourse/ci/blob/9663b91a39fd45f075df78283d227dfb4aed9d97/pipelines/reconfigure.yml
[`set_pipeline`]: https://github.com/concourse/rfcs/pull/31
[ci.concourse-ci.org]: https://ci.concourse-ci.org


### yet another RBAC

3. Not having our API described as Kubernetes resources makes those running
   Concourse on Kubernetes to not be able to leverage already-existing RBAC
   mechanisms that might already have been setup for teams that use a given
   cluster.



# Proposal

1. Define Concourse's custom resources

- i. `Pipeline`: definition of what a pipeline is

This is pretty much what @jchester has already demonstrated to work:
https://github.com/jchesterpivotal/knative-build-pipeline-poc.

e.g.

```yaml
apiVersion: concourse-ci.org/v1alpha1
kind: Pipeline
metadata:
  name: mypipeline
  namespace: myteam-ns
spec:
  team: myteam
  config:
    resources:
    - name: repository
      type: git
      source: {uri: https://github.com/concourse/concourse}
    jobs:
    - name: successful-job
      public: true
      plan:
      - get: repository
      - task: job-task
        file: repository/ci/task.yml
```

As pipelines are configured on a per-team basis, and Kubernetes tenancy is
usually granted on a per-namespace basis, a second construct would be needed:
`Team`.


- ii. `Team`: team authorization configuration

Defines the mapping between a given set of teams and a namespace.


```yaml
apiVersion: concourse-ci.org/v1alpha1
kind: TeamsBindings
metadata:
  name: myteam
  namespace: concourse-ns
spec:
  teams:
    - name: myteam
      namespace: myteam-ns
```

This way we allow a pipeline that has been set up in a particular namespace to
be set against a particular Concourse team.


- iii. `JobTrigger`: request to running a specific job of a pipeline

Having a way of programatically triggering jobs of a pipeline would allows those 


2. Implement the reconciliation functionality to get those applied

Have Concourse's [`atc`] implementing controllers that take care of
reconciliating the desired state expressed as Kubernetes resources (e.g.,
`Pipeline` kind) with Concourse's state (pipeline config).

[`atc`]: https://concourse-ci.org/architecture.html#component-atc


To achieve these, we could either go with the controller being embedded in `atc`
itself, having access to the database directly, or have it outside of `atc`,
making requests to it via the API.

In the second case (outside `atc`), we could build upon the work that has been
done for getting [`client_credentials` grant] in.


# Open Questions

- **How accesss should be controlled?**

Building upon [don't use `concourse` tokens], we can have special 


[`client_credentials` grant]: https://github.com/dexidp/dex/pull/1629
[don't use `concourse` tokens]: https://github.com/concourse/concourse/issues/2936


- **Would this "deprecate" the [OPA RFC]**?

I don't think so - I think there's a place for both.

Going with the approach of keeping the resource definitions at a very high level
(pipelines, teams, and job runs), there wouldn't be much that an OPA looking
only at kubernetes objects could do with regards to more "dynamic" data that
Concourse deals with (e.g., task configs that get dynamically discovered).


[OPA RFC]: https://github.com/concourse/rfcs/pull/41


# Answered Questions



# New Implications


