# Terms
- **cache object** a BLOB and relevant metadata that Concourse needs to persist. These could be Resource Caches, Task Caches or Build Caches.
- **worker** Concourse executes steps on a **worker** and implements some **worker** interface. Concourse is agnostic of the runtime representation of the worker (eg. K8s pod, node or cluster).

# Summary

After spiking on a few solutions for storage on Kubernetes our recommendation is to use an image registry to store **cache objects** for steps.

# Motivation

As we started thinking about the Kubernetes runtime we realized that we need to think about what our storage solution would be before proceeding with any other part of the implementation. Storage has a huge effect on how Concourse interacts with the runtime (Kubernetes). Storage also had a lot of unknowns, we didn't know what the storage landscape on Kubernetes looked like and what options were available to us. Storage also has a huge impact on the performance of the cluster, in regards to storage and initialization of steps.

## Requirements
An ideal storage solution can do the following :

- image fetching from the CRI k8s is using
- transfer **cache objects** between steps (whatever represents a step, most likely a pod)
- cache for resources and tasks
- stream **cache objects** across worker runtimes (k8s worker sends artifact to garden worker)

## Criteria
- security
- performance, aka initialization time (time spent running workloads on a single k8s worker, as well as across workers)
- resource usage to run this storage solution

# Proposal

**TL;DR**: We recommend going with the image registry option because it satisfies all the requirements and gives us a bunch of options to improve performance when compared to the blobstore option. It also provides a very flexible solution that works across multiple runtime workers. [See Details](#image-registry-to-store-artifacts)

Furthermore, the CSI is a useful interface for building the storage component against. [See Details](#csi)

# Storage Options considered
## Baggageclaim Daemonset 
### Description
A privileged baggageclaim pod would manage all the **cache object** for step pods. The pod can be provided sufficient privilege to create overlay mounts using `BiDirectional` value for `mountPropagation`. The `volumeMount` object allows specifying a volume `subPath`.

This approach didn't work using GCE PDs or vSphere Volumes ([Issue](https://github.com/kubernetes/kubernetes/issues/95049)). It does work using `hostPath` option, however, that would require a large root volume and wouldn't be able to leverage IaaS based persistent disks. 

The pod would run on all nodes that Concourse would execute steps on.
 
### Pros
+ Leverage baggageclaim
	+ volume streaming between nodes would work using the current Concourse architecture
	+ resource and task caches would also work using the current Concourse architecture
	+ would be able to stream **cache objects** across worker runtimes as it would be mediated via the web
+ Concourse would have complete control over volume lifecycle
+ would have negligible overhead for steps scheduled on the same node as no input/output stream would be required

### Cons
- Not being able to use IaaS based persisent disks doesn't offer a viable solution. K8s nodes would need to have large root volumes.
- Wouldn't have support for hosting images by default. However, `baggageclaim` could be extended to add the APIs
- `baggageclaim` itself doesn't have any authentication/authorization or transport security (https) mechanisms built into it

## Image Registry to store artifacts
### Description
Each **cache object** is represented as a image layer for a repository in an image registry. [SPIKE using registry to store artifacts](https://github.com/concourse/concourse/issues/3740). Concourse would require a managed image registry as a dependency. For each step, Concourse would generate a image config and manifest with all the relevant inputs modeled as image layers. 

### Pros
- Would have support for building an image in a step and using it as the image for a subsequent step. This would require the image registry to be accessible by the CRI subsystem on a node
- Image registries are are critical to operating on K8s and as such there are plenty of options for leveraging managed IaaS based solutions such as GCR, ECR, ACR to on prem solutions like Harbor. Therefore, it would be a safe assumption that a Concourse on K8s user would already have a registry available for use.
- Could explore further de-coupling by exploring [csi-driver-image-populator](https://github.com/kubernetes-csi/csi-driver-image-populator) when using registries for storing artifacts. Listed as a sample driver in the CSI docs and README says it is not production ready. Last commit was Oct 2019. There is also another utility - [imgpack](https://github.com/k14s/imgpkg) which allows arbitrary data store in images as layers.
- Leverage performance enhancements to registries such as [pull through cache](https://docs.docker.com/registry/recipes/mirror/)
- Use a standardized and documented [OCI image-spec protocol](https://github.com/opencontainers/image-spec)
- LRU based local caching of image layers by the K8s CRI
- Established ways of securely pushing/pulling blobs from an image registry
- As this would be a centralized storage solution
	- it doesn't impact what a K8s based Concourse worker looks like
	- Simplified GC
	- Would support streaming across worker runtimes
	
### Cons
- Some registries such as GCR don't expose an API to delete layers directly
- **cache object** would have to have a fixed static path in the image file system to be able to reuse the same layer. This would require some additional handling on Concourse to support [input-mapping](https://concourse-ci.org/jobs.html#schema.step.task-step.input_mapping) and [output-mapping](https://concourse-ci.org/jobs.html#schema.step.task-step.output_mapping)
- Adds extra development overhead to generate new image config & manifests to leverage **cache object** layers
- Adds extra initialization overhead. Concourse wouldn't have control over the local caches on K8s nodes, so volumes would always have to be pushed to the centralized registry and pulled at least once when executing a step
- Potentially adds substantial load on registry, as Concourse would be creating a new file system layer for every **cache object**
- There isn't a well documented approach to setup an in-cluster secure registry. The setup requires exposing an in-cluster registry externally with traffic routed via an LB. [Prior spike](https://github.com/concourse/concourse/issues/3796)

## S3 Compatible Blobstore
## Description
Each **cache object** is stored in a blobstore. Concourse would require a mananaged blobstore as a dependency. For each step, Concourse would pull down the relevant blobs for inputs and push blobs for outputs. 

### Pros
- Scale well (GCR uses GCS as the underlying storage)
- Could explore further de-coupling by exploring CSI driver
- Established ways of securely fetching/pushing blobs from an a blobstore
- As this would be a centralized storage solution
	- it doesn't impact what a K8s based Concourse worker looks like
	- Simplified GC
	- Would support streaming across worker runtimes
	
### Cons
- Wouldn't have support for hosting images by default.
- Adds another dependency for Concourse (depending on where Concourse is deployed there might be managed solutions available) 
- Lack of standardized APIs
- Adds extra initialization overhead. Concourse wouldn't have a local cache, so volumes would always have to be pushed & pulled for steps
- Concourse would potentially be heavy user of the blobstore

## Baggageclaim + CSI Implementation
### Description
TODO
### Pros
TODO
### Cons
TODO

## Persistent Volumes
Each **cache object** would be stored in its own persistent volume. Persistent volume snapshots would be used to reference **cache object** versions.

### Pros
- Would leverage native k8s offering
- Maps well to Concourse's use of **cache objects** and offloads the heavy lifting to K8s
- Potentially wouldn't require volumes to be streamed at all

### Cons
- Wouldn't have support for hosting images by default.
- IaaS based limits on [volume limits per node](https://kubernetes.io/docs/concepts/storage/storage-limits/#dynamic-volume-limits) prevents this from being a scalable solution
- CSI Snapshotting feature is optional and not every driver supports it ([Drivers & features they support](https://kubernetes-csi.github.io/docs/drivers.html#production-drivers))
- As this would NOT be a centralized storage solution, it wouldn't support workers across multiple runtimes or even K8s clusters

## K8s POC (Baggagelciam peer-to-peer)
Each step would have a sidecar container to populate input **cache objects** and host outputs **cache objects** via an HTTP API.`beltloader` is used to populate inputs. `baggageclaim` is used to host outputs. `baggageclaim` was also modified to allow **cache objects** to be accessed via the registry APIs (support images).

### Pros
- No external dependencies are required
- Supports worker-to-worker streaming bypassing Concourse web

### Cons
- the `step` pod's lifecycle is tied to the **cache object** lifecycle (pods have to be kept around until the **cache object** they host is required). This would increase the CPU & memory usage of a cluster. 
- there isn't a simple mechanism to allow the k8s container runtime to securely access the `baggageclaim` endpoints to fetch images
- As this would NOT be a centralized storage solution, it would require exposing the `baggageclaim` endpoints via `services` to be accessed externally
- `baggageclaim` itself doesn't have any authentication/authorization or transport security (https) mechanisms built into it

# Other considerations
## CSI 
The [Container Storage Interface](https://github.com/container-storage-interface/spec/blob/master/spec.md) provides a generic interface for providing storage to containers.

CSI was developed as a standard for exposing arbitrary block and file storage storage systems to containerized workloads on Container Orchestration Systems (COs) like Kubernetes. With the adoption of the Container Storage Interface, the Kubernetes volume layer becomes truly extensible. Using CSI, third-party storage providers can write and deploy plugins exposing new storage systems in Kubernetes without ever having to touch the core Kubernetes code. This gives Kubernetes users more options for storage and makes the system more secure and reliable. [Source](https://kubernetes.io/blog/2019/01/15/container-storage-interface-ga/#why-csi)

The CSI spec can be used to wrap every solution listed above. It provides an API through which the chosen solution would be consumed.

### Pros
- Can be deployed/managed using k8s resources ([hostPath CSI Driver example](https://github.com/kubernetes-csi/csi-driver-host-path/blob/master/docs/deploy-1.17-and-later.md))
- Allows the storage mechanims to be swapped more easily
	- can be an extension point for Concourse community
- De-couples Concourse from its usage of storage
	- the driver could be patched/upgraded indepdently of Concourse 
- The CSI Spec is quite flexible and has a minimum set of required methods (the other set of features are opt-in)
- CSI supports multiple deployment topologies (master, master+node, node)
- Provides a scheduling extension point for volume aware scheduling

### Cons
- extra overhead for development, packaging and deployment
- the CSI version may be tied to a K8s version

## Fuse
This might simplify our usage of external storage solutions such as blobstores. There isn't a supported solution in K8s at the moment. However, this would be something worth considering if that were to change. [Click here to view the current issue requesting K8s development](https://github.com/kubernetes/kubernetes/issues/7890).

# Open Questions

- Do we implement our own version of the csi-image-populator?
- Should we implement this as a CSI driver?


# Answered Questions


# Related Links
- [Storage Spike](https://github.com/concourse/concourse/issues/6036)
- [Review k8s worker POC](https://github.com/concourse/concourse/issues/5986)


# New Implications

Will drive the rest of the Kubernetes runtime work.
