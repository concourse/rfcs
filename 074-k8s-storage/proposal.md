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

**TL;DR**: After [spiking on the CSI driver interface](https://github.com/concourse/concourse/issues/6133) we now recommend creating a CSI driver based on baggageclaim.

Furthermore, the CSI is a useful interface for building the storage component against. [See Details](#csi)

## Level Setting

Before getting into the meat of the proposal let's first understand level set on our understanding of [baggageclaim]() and the [CSI spec]().

### What does Baggageclaim do?

Baggageclaim comes as two components: a client and a server communicating over an HTTP REST API. The server component manages volumes within a specified directory on the host.
Volumes are created based on one of three strategies:
- Empty Strategy: creates an empty volume
- COW Strategy: creates a volume based on an existing volume
- Import Strategy: creates a volume based on a local directory or tar ball

Baggageclaim keeps tracks of all volumes by querying the filesystem structure, therefore no database component is needed. Volumes are assigned an ID that is passed in by the baggageclaim client when the `CreateVolume` request is made.

The [HTTP REST API](https://github.com/concourse/baggageclaim/blob/ea9252e4fcca101f32971cfb5ff47c3355c7c91e/api/handler.go#L26-L38) allows a baggageclaim client to:
- Create volumes with one of the above strategies
- Destroy volumes
- Query for a list of all volumes
- Query for the properties of a single volume
- Stream the contents of a volume

Supports multiple filesystem drivers (overlay, btrfs, naive\*). Overlay is the recommended driver to use as it's the most stable. All drivers support all features of baggageclaim.

\* _naive simply `cp`'s files into new directories and isn't really a "filesystem"_

### What's a CSI Driver?

In a container orchestration (CO) system, such as Kubernetes or Cloud Foundry, you need a way to provide storage to containers. This could be ephemeral (lifecycle is tied to the container) or persistent (operates outside of the container's lifecycle) storage. In order to support many different storage providers the [CSI spec](https://github.com/container-storage-interface/spec/blob/master/spec.md) was made to allow CO's to have a consistent way to ask for storage for containers.

A CSI driver is an implementation of a gRPC interface. The CO communicates with a CSI driver over a unix domain socket.

A CSI driver is made up of two components, both serving different parts of the CSI's gRPC interface:

**Controller Plugin**: Can be run anywhere. Serves the **Controller and Identity Service**.

**Node Plugin**: Must be run on the Node where the storage requested by the CO is to be provisioned. Serves the **Node and Identity Service**. (Yes, the indentity service is served by both plugins).

The full list of functions for the interface are available in the [CSI Spec](https://github.com/container-storage-interface/spec/blob/master/spec.md#rpc-interface). There is some flexibility as to how you architect your CSI driver. The [CSI spec has some examples](https://github.com/container-storage-interface/spec/blob/master/spec.md#architecture). The volume lifecycle the CSI driver is expected to follow is also [diagramed in the CSI Spec](https://github.com/container-storage-interface/spec/blob/master/spec.md#volume-lifecycle).

When implementing a CSI driver for Kubernetes it is helpful to understand when certain CSI functions are called. CSI functions are typically called after the creation/modification of some Kubernetes API objects. It's important to note here that a CSI driver **does not know anything about kubernetes API objects**. In order for CSI functions to be called at the right time a CSI driver depends on various sidecar containers that monitor for certain Kubernetes Storage objects. These sidecars are provided by the Kubernetes team; a list of these ["helper containers" is available in this Kubernetes CSI Design document](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/container-storage-interface.md#recommended-mechanism-for-deploying-csi-drivers-on-kubernetes).

[The CSI Driver Spike](https://github.com/concourse/concourse/issues/6133#issuecomment-708471004) contains some notes that show which CSI functions are called when certain Kubernetes objects are created.

## Proposed Implementation of Baggageclaim as a CSI Driver

Targeting Kubernetes Version 1.19

Follow the recommended deployment strategy from the Kubernetes team [described in this design document](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/container-storage-interface.md#recommended-mechanism-for-deploying-csi-drivers-on-kubernetes) with the following differences:
- no `external-resizer` container. Not planning to support resizing.
- no `external-snapshotter` container. We will use the `CLONE_VOLUME` feature to create COW volumes in baggageclaim instead of trying to use snapshots.
- An extra volume must be mounted for each replica Pod in the DaemonSet. This volume, which should be very large, will be used by baggageclaim to store the volumes that it creates on each Kubernetes node.
- We plan to **not guarantee** the requested storage capicity because we have no idea how much space any given step in Concourse will use. Kubernetes will force us to specify a storage request but our CSI driver will ignore this value. This goes against the CSI spec.

Let's go over some use cases to get an understanding about how the implementation may work.

### Creating An Empty Volume

A user creates a PVC:
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: volume-guid # The volume ID that Concourse Web keeps track of
spec:
  accessModes:
    - ReadWriteOnce # the only accessMode we will support
  volumeMode: Filesystem
  resources:
    requests:
      storage: 1Gi # can be any value, we ignore this
  storageClassName: baggageclaim
```

The [`external-provisioner`](https://github.com/kubernetes-csi/external-provisioner) will call `Controller.CreateVolume`. In this case `CreateVolume` will generate an ID for tracking the volume.

With the PVC "created" (from the perspective of Kubernetes), a user can now reference the PVC in a Pod.

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: output
spec:
  containers:
  - name: output
    ...
    volumeMounts:
      - name: artifact-name
        mountPath: /tmp/artifact-name
  volumes:
    - name: artifact-name
      persistentVolumeClaim:
        claimName: volume-id
```

`Controller.PublishVolume` will get called. This will be a no-op.

`NodeStageVolume` will get called. This will be a no-op.

`NodePublishVolume` will get called. Baggageclaim will create a volume based on the `EmptyStrategy`.

```go
volume, err := ns.bagClient.CreateVolume(ns.logger, req.VolumeId, baggageclaim.VolumeSpec{
    Strategy:   baggageclaim.EmptyStrategy{},
    Properties: map[string]string{},
})
```
_This could also be done in `NodeStageVolume`_

Still in `NodePublishVolume`, the volume will then be mounted at the path provided in the `NodePublisVolumeRequest`:

```go
mounter := mount.New("")
path := volume.Path()
targetPath := req.GetTargetPath()
options := []string{"bind"}
glog.V(4).Infof("concourse: mounting baggageclaim volume at %s", path)
if err := mounter.Mount(path, targetPath, "", options); err != nil {
    return nil, err
}
```

The volume has been successfully provided to Kubernetes by this point.

### Creating A Cloned Volume

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: volume-guid
spec:
  accessModes:
    - ReadWriteOnce
  volumeMode: Filesystem
  resources:
    requests:
      storage: 1Gi
  storageClassName: baggageclaim
  dataSource: # will only support cloning other baggageclaim volumes
    name: some-other-pvc
    kind: PersistentVolumeClaim
```

The [`external-provisioner`](https://github.com/kubernetes-csi/external-provisioner) will call `Controller.CreateVolume`. In this case `CreateVolume` will see that a `VolumeContentSource` has been provided and will pass along the source volume's ID to later requests using the `Volume.volume_context` field.

```go
if req.GetVolumeContentSource() != nil {
    volumeSource := req.VolumeContentSource
    switch volumeSource.Type.(type) {
    case *csi.VolumeContentSource_Volume:
        if srcVolume := volumeSource.GetVolume(); srcVolume != nil {
            volumeContext["sourceVolumeID"] = srcVolume.GetVolumeId()
        }
    default:
        status.Errorf(codes.InvalidArgument, "%v not a proper volume source", volumeSource)
    }
}
```

`Controller.PublishVolume` will get called. This will be a no-op.

`NodeStageVolume` will get called. This will be a no-op.

`NodePublishVolume` will get called. Baggageclaim will create a volume based on the `COWStrategy`, fetching the parent volume from `VolumeContext`.

```go
id, _ := volumeContext["sourceVolumeID"]
sourceVolume, _, _ := ns.bagClient.LookupVolume(ns.logger, id)

volume, err := ns.bagClient.CreateVolume(ns.logger, req.VolumeId, baggageclaim.VolumeSpec{
    Strategy:   baggageclaim.COWStrategy{Parent: sourceVolume},
    Properties: map[string]string{},
})
```
_This could also be done in `NodeStageVolume`_

Still in `NodePublishVolume`, the volume will then be mounted at the path provided in the `NodePublisVolumeRequest`. The volume, populated with data from the parent PVC, has been successfully provided to Kubernetes by this point.

### Streaming Volumes Inside A Kubernetes Cluster

### Streaming Volumes To An External Baggageclaim

# Open Questions

- When we need to have a volume available on multiple k8s nodes, how do we do this in a baggageclaim CSI driver?
  - Would it make sense to support `ReadWriteMany` as the volume's `accessMode` instead of `ReadWriteOnce`?
- What does the Concourse database model for volumes look like with a k8s worker running a baggageclaim CSI driver?
- How will the CSI driver stream a volume between k8s nodes?
  - What is the recommended way for a CSI controller to maintain state and know which volume is on which node(s)?
  - How will we stream single files in a volume? (i.e. when Concourse needs to read a task config from the artifact of a get step)
- What is the recommended way to deploy the CSI driver? (e.g. StatefulSet, DaemonSet, etc.) _StatefulSet appears to fit our usecase best_
- For volume streaming, should we go for the in-cluster P2P solution or stick with streaming through the Concourse web nodes?
- Does the CSI driver need to be aware of each Concourse cluster that is using it? Another way of phrasing this question: can/should the CSI driver support multiple concourse installations? Do we need to do anything special to support this if we decide yes?


# Answered Questions

- Should we implement this as a CSI driver? **Yes we do after doing the CSI Driver POC Spike**
- Do we implement our own version of the csi-image-populator? **Yes but based on baggageclaim instead of image layers**

# Related Links
- [Storage Spike](https://github.com/concourse/concourse/issues/6036)
- [Review k8s worker POC](https://github.com/concourse/concourse/issues/5986)
- [CSI Driver POC Spike](https://github.com/concourse/concourse/issues/6133)


# New Implications

Will drive the rest of the Kubernetes runtime work.

---

# Appendix - Alternative Storage Options considered

The follow are some of the other storage on Kubernetes options that we considered.

## Baggageclaim + CSI Implementation
### Description
A privileged baggageclaim pod would manage all the **cache object** for step pods. The baggageclaim pod can be provided sufficient privilege to create overlay mounts and have those mounts propagate back to the host using the `BiDirectional` value for `mountPropagation`.

The pod would run on all nodes that Concourse would execute steps on.
 
### Pros
+ Leverage baggageclaim
	+ volume streaming between nodes would work using the current Concourse architecture
	+ resource and task caches would also work using the current Concourse architecture
+ Web can manage stoage via native k8s stoage objects
  + Concourse would have complete control over volume lifecycle
  + Operator can query k8s api to observe all volumes
+ would have negligible overhead for steps scheduled on the same node as no input/output stream would be required
+ Disk where volumes are managed can be backed by any other CSI driver (as a long as baggageclaim can make overlay mounts on it)
  + Can leverage tools in k8s to manage the disk that baggageclaim is writing to
### Cons
- CSI drivers are meant to guarantee storage capacity; baggageclaim does not currently do this, it provides unbounded disk space
  - This CSI driver will not be meant for usage outside of Concourse
- Wouldn't have support for hosting images by default. However, `baggageclaim` could be extended to add the APIs
  - Crazy Idea 1: somehow load the image into the CRI that's running on the node
  - Crazy Idea 2: Can a volumeMount override the root path (`/`)?
- `baggageclaim` itself doesn't have any authentication/authorization or transport security (https) mechanisms built into it
  - k8s has networking tools that we can leverage to ensure only authorized clients can talk to it

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

