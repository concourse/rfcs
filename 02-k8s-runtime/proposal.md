# Summary
This RFC outlines the concerns and considerations involved in introducing a
Kubernetes (K8s) runtime, which schedules K8s Workloads in place of Garden
Containers and (potentially) Baggageclaim Volumes.

# Proposal
The initial approach should work towards a common "Orchestrator" interface, with
a clear contract for the following :
- RunTask
- CheckResource
- GetResource
- PutResource

Currently these are all actions which the ATC's scheduler performs to eventually
create Garden containers and Baggageclaim Volumes. The database representation,
and underlying  implementation is unified in the `worker` package, but there are
other corners of the ATC codebase where objects are decorated with additional
database context.

TODO:
- Summary of Proposed Changes
- Overarching Changes
- Changes to ATC
- Detail the impact of new Concourse Worker registration and GC
- Caveats - K8s Cluster Operator Needs
- Open Q's - Concourse Worker per K8s Node

# Clarification of Terminology
There's some terminology clash between Concourse and K8s concepts (eg. Job,
Worker). The list below serves to disambiguate and clarify the use of terms
throughout this document.

**Concourse Worker**  : A process running on a host (VM, K8s Container, etc)
which registers a worker with a Concourse cluster. Can include Garden or
Baggageclaim, but these may be optional.

**K8s Node** : "A node is a worker machine in Kubernetes"; referred to
hereinafter as "K8s Node" rather than "Worker Node" to avoid confusion with
Concourse Workers.  ( see: [Glossary - Node][node] )

**K8s Job** :  
K8s Job - "A finite or batch task that runs to completion" - ( see:
[Glossary - Job][k8s-job] )

**K8s Workload** : An umbrella term referring to the various types of
 object which can be created by a Kubernetes Cluster. ( see:
 [Glossary - Workloads][workloads] )

**Baggageclaim Volume** : A directory on a file-system managed by Baggageclaim,
used by Concourse to store Resource Caches, Task Caches, and Image Resource
Caches.

**K8s Volume** : "A directory containing data, accessible to the containers in a
 pod."; referred to hereinafter as "K8s Volume" to disambiguate K8s Volumes and
Baggageclaim Volumes.



[node]: https://kubernetes.io/docs/reference/glossary/?all=true#term-node
"K8s Glossary - Node"
[k8s-job]: https://kubernetes.io/docs/reference/glossary/?all=true#term-job
"K8s Glossary - Job"
[workloads]: https://kubernetes.io/docs/concepts/ "K8s Glossary -
 Workloads"

# Summary of Proposed Changes

## Overarching Changes

## Changes to ATC

## Changes to Worker Registration

# Open Questions

## Volume Management

The POC on the `wip-feature/2037` branch runs a single Concourse Worker in a Pod
with a Baggageclaim server. Each step in a Concourse Job streams Baggageclaim
Volumes using a CLI and Init Containers to move files into mounted `EmptyDir`
K8s Volumes.

Get, Put and Task steps require different combinations of inputs and outputs
depending on users' pipelines. The contract with Concourse users is that
inputs will be mounted as a Copy(OnWrite) of the Resource Cache Volume, and
modifications to the input within a task will not mutate the contents of the
cache.

Is there a more "K8s way" of managing volumes between Pod executions, which
provides and interface the ATC can leverage to manage volumes similar to how
Baggageclaim Volumes are managed. Could a replica set place a worker on each K8s
Node and use `HostPath` Volumes or create Persistent Volumes to store Resource
Caches, Task Caches, and Image Resource Caches, etc.?

## The Use of K8s Jobs

K8s Jobs provide a useful native K8s abstraction for Concourse Build Plan Steps.
The `JobSpec` can indicate that a K8s Job should run to completion only once,
which is what Concourse wants of each Build Step.

Could a CustomResourceDefinition (CRD) be used to represent the Containers
created for Concourse Tasks and Resources? This would allow a user or operator
to easily recognize and differentiate Concourse Containers and their
corresponding workloads from other workloads on the K8s Cluster.

Would a K8s Controller be used to create Pods, Jobs, etc. and manage the GC of
these resources on the cluster, as well as their placement on specific nodes?

## Preferring `image_resource` to `rootfs_uri`

Typically the recommendation to users to define container images for Tasks is
to use `image_resource`, and typically the type used is the `docker-resource`.

The Concourse docs state:
> "`image_resource` is a preferred way to specify base image and `rootfs_uri` is
> not recommended."

but in K8s-land, specifying a Container Image repository is the only way to
create a container, and K8s will cache these images. In order to support this
as the preferred way to define container images, we will need to find a viable
solution which saves the exported contents from `image_resource` to the K8s
registry.

# New Implications

## Multiple Worker Orchestrator "Types"

Operators of a Concourse cluster may need to be aware of the underlying
orchestrator their cluster is using, or the "type" of registered workers.

## Worker Tagging

Tagging of Concourse Jobs for specific workers might need to change to
accommodate K8s Pod `nodeSelector`s which allow users to select specific K8s
Nodes schedule the necessary Workloads on. Affinity and anti-affinity also
impact the selection of a K8s Node for a given Pod, and users may wish to use
these features similar to how tagging currently works in Concourse.

See ["Assigning Pods to Nodes"][nodeSelector]

  [nodeSelector]: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#nodeselector ""Assigning Pods to Nodes""
