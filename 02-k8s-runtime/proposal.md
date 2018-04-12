# Summary
This RFC outlines the concerns and considerations involved in introducing a
Kubernetes runtime, which schedules Kubernetes Resources in place of Garden
Containers and (potentially) Baggageclaim Volumes.

# Proposal
The initial approach should work towards a common "Orchestrator" interface, with
a clear contract for the following :
- RunTask
- CheckResource
- GetResource
- PutResource

Currently these are all actions which the ATC's scheduler performs to eventually
create Garden containers and Baggageclaim volumes. The database representation,
and underlying  implementation is unified in the `worker` package, but there are
other corners of the ATC codebase where objects are decorated with additional
database context.

TODO:
-

# Summary of Proposed Changes

## Overarching Changes

## Changes to ...

# Caveats

# Open Questions

## Volume Management

The POC on the `wip-feature/2037` branch uses Baggageclaim on a K8s 'worker'.
The Worker runs a Baggageclaim server in a Pod, and each step in a job streams
volumes using a Baggageclaim cli and Init Containers to move files into mounted
`EmptyDir` K8s Volumes.

Get, Put and Task steps require different combinations of inputs and outputs
depending on the implementation. The contract with Concourse users is that
inputs will be a Copy(OnWrite) of the Resource Cache Volume, and modifications
to the input within a task will not mutate the contents of the cache.

Is there a more "K8s way" of managing volumes between Pod executions, which
provides and interface the ATC can leverage to manage the state of volumes
similar to what Baggageclaim provides.

## The Use of K8s Jobs

Kubernetes' Jobs resource provides a useful native K8s abstraction for Concourse
Build Steps. The `JobSpec` can indicate that a Job should run to completion
only once, which is what Concourse wants of each Build Step.

## Preferring `image_resource` to `rootfs_uri`

Typically the recommendation to users to define container images for Tasks is
to use `image_resource`, and typically the type used is the `docker-resource`.

The Concourse docs state
> "`image_resource` is a preferred way to specify base image and `rootfs_uri` is
> not recommended."
but in K8s-land, specifying a Container Image repository is the only way to
create a container, and K8s will cache these images. In order to support this
as the preferred way to define container images, we will need to find a viable
solution which saves the exported contents from `image_resource` to the K8s
registry.

# New Implications
