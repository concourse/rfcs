# How Can Concourse Best Leverage K8s as a Runtime?
The current landscape of Kubernetes for CI/CD leaves us with a lot of new
options since the original Kubernetes Runtime RFC (#2) and it makes sense for
the Concourse team + community to have a discussion around how we can best
leverage K8s for Concourse workloads.

The goal of this proposal is to find a path forward and drive towards a POC
which assesses the many options available to us, and evaluate the benefits of using 
[Tekton](https://cloud.google.com/tekton/) or rolling our own solution using
native K8s primatives to run Concourse workloads on Kubernetes. 

## What Do We Hope to Gain?
### Simpler Deployment and Operations 
Ideally relying on K8s would mean that operators only need to deploy the ATC(s)
and point them at a Kubernetes cluster, and they would no longer need to worry
about deploying and managing workers.

### Scheduling Efficiency
A lot of the current work in the [Runtime
backlog](https://github.com/orgs/concourse/projects/23) is focused on
efficiently using the pool of workers that are available; we'd eventually like
to take into acccount the available compute resources on a worker as part of
the Scheduling

### Get Hype?
🚂  CHOO CHOO! 🚃  AIN'T NO BRAKES ON THE K8S HYPE TRAIN!! 🚃 ALL ABOARD!!!! 🚃

# Technical Concerns
Regardless of how we choose to move forward leveraging K8s, there's some common
concerns which we'll need to address to make the K8s runtime a reality (
whether we use Tekton or roll our own solution leveraging K8s primatives )

## Unit of Execution
*tldr:* Does the proposed K8s runtime schedule individual steps, builds (with
many steps), or a whole pipeline at once?

What is the smallest unit of exectution we can / want to leverage in the K8s
runtime?  Ensure we're optimially leveraging the features of Kubernetes and
addressing any technical challenges around volume streaming between steps of a
build.

## Image Support
*tldr:* How does the proposed runtime support existing task image behaviour in
Concourse?

There are currently many different ways in which images are fetched or
specified by users to define the RootFS for a task step or custom resource.
Users can provide an `image_resource` which uses a resource `get` operation to
fetch the image RootFS into a volume, but they can also use an output from
another step in the pipeline as the `image:` for a task. 

## Volume Caching
*tldr:* Where does the existing behaviour of using Bagggageclaim to cache
volumes (resource caches, task caches, etc) fit in, given we cannot control
Volumes' lifecycle outside of a pod / deployment ?


# Tekton
https://github.com/tektoncd

## Unit of Execution
 


## Image Support
From https://github.com/tektoncd/pipeline/issues/639 it seems like image building is taking
some precident in Tekton to allow users to build an image and use it for a later Task.

## Volume Caching

https://github.com/tektoncd/pipeline/blob/master/docs/developers/README.md#how-are-resources-shared-between-tasks

`PipelineRun` supports using a [Persistent Volume
Claim](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#persistentvolumeclaims)
or a GCS bucket to share artifacts between tasks.

# Rolling our Own
Using K8s Primitives, or some mix of K8s primatives and custom CRDs. 

## Unit of Execution
One Pod for a running build's steps? This would make sharing volumes between
steps a lot simpler, but we may need to continually update the pod as steps are
reached over the course of a build's execution.

Jobs for build steps?
> K8s Jobs provide a useful native K8s abstraction for Concourse Build Plan
> Steps. The JobSpec can indicate that a K8s Job should run to completion only
> once, which is what Concourse wants of each Build Step.

## Image Support
We'd need to determine how to use an abitrary RootFS as an image for whatever
containers we're scheduling in Pods, deployments, Jobs, etc. This might involve
importing the image into the local registry and using it later on, which also
implies that the K8s runtime will need to change how it thinks about images. 

Images would no longer be cached in Volumes the same way outputs / get steps
are, and would be cached by the runtime in the registry itself. At this point
most tasks would benefit more from being able to specify the image repository
directly, rather than using the current preffered method of `iamge_resource`.


## Volume Caching
Separate Baggageclaim component deployed to support this? One Baggageclaim per
node using anti-affinity? How do we mounty volumes to the containers of a build
step.

K8s has some GC concepts which may help us to model the ownership and retention
of pods (and their corresponding volumes), which will be required for leter
steps

https://kubernetes.io/docs/concepts/workloads/controllers/garbage-collection/

Suggestions from the OG K8s RFC are to use Local Persistent Volumes or a local
object store (eg Minio).
- https://github.com/concourse/rfcs/pull/2#discussion_r198704652
- https://github.com/concourse/rfcs/pull/2#issuecomment-403347847

We might be reinventing a wheel which Tekton has already invented if we go down
either of these paths, so its important to consider whether their wheel is
sufficient, or whether we can mold it to Concourse's needs.
