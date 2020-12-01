# Summary

An overview of how we plan to implement Kubernetes as a runtime for Concourse.

# Motivation

We want to leverage Kubernetes as a runtime for container orchestration. The K8s runtime would be specifically leveraged for;

- `stability` by offering more robust container placement 
- `cost savings` by offering simpler mechanisms for scaling the system up/down to match demand
- `security` by offering defined interfaces for auditing and policy management
- `observability` by leveraging logging & metrics solutions in the K8s ecosystem

# Terms
* **K8s Worker Client** The K8s implementation of the [worker.Client](https://github.com/concourse/concourse/blob/master/atc/worker/client.go#L31)
* **Worker Lifecycle Component** Is responsible for registering, heartbeating, volume and container garbage collection
 
# Proposal
## Worker Mapping
A K8s Concourse worker would be represented by a K8s worker + K8s namespace. This was the mapping suggested in the [k8s POC](https://github.com/concourse/concourse/issues/5209), where a [namespace](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/) in a [cluster](https://kubernetes.io/docs/concepts/architecture/) represented a single Concourse worker.

This leverages multi-tenant nature of Kubernetes and allows the Kubernetes cluster operator to manage and isolate Concourse workloads via the targeted namespace. It also allows an operator to configure capacity for a Concourse worker using [Resource Quotas](https://kubernetes.io/docs/tasks/administer-cluster/manage-resources/quota-memory-cpu-namespace/).

With this mapping a single Kubernetes cluster can represent multiple workers and manage resources based on namespaces.

### Worker Mapping Refactor

Within the code base we have the following objects and interactions going on:
- A `worker.Pool that returns a `worker.Worker` (garden + baggageclaim clients)
- The worker pool is passed into the WorkerClient which is the thing that executes a step in a build plan. It does this by asking the worker pool to select a worker and then uses that worker to carry out the workload
- `worker.Client` is then passed into the engine.

The way these objects are nested make it difficult for us to add a runtime, which is evident in Ciro's POC work where he simmply commented out the worker pool code.

Instead the code should:
- The engine should recieve a `worker.Pool` instead of `worker.Client`. engine would then have the `worker.Pool` select a worker and return a `worker.Client`
- `worker.Worker` will be added to `worker.Client` and removed from `worker.Pool`
- `worker.Pool` will return a `worker.Client` instead of `worker.Worker`

This results in the scheduling decision being made one layer up from where it's currently made and should make it easier for us to add runtimes like Kubernetes or even Nomad.

## Authenticating to the k8s worker
Concourse would support both mechanisms for authenticating to the k8s cluster

* kubeconfig
* service account

The `kubeconfig` option provides a mechanism for providing access to workers across clusters and for running web locally targetting a k8s cluster.
The `service account` option would be useful for in-cluster.
deployments of web targetting the cluster it was deployed on.

## Boundary Where We Introduce Kubernetes Logic
Same as the [k8s POC](https://github.com/concourse/concourse/issues/5209), implement the Kubernetes worker behind the [`worker.client`](https://github.com/concourse/concourse/blob/master/atc/worker/client.go).

### Storage
The k8s runtime will continue to use baggageclaim to provide volumes to containers. This will be provided by creating a Baggageclaim CSI Driver . [See RFC 74 for more details](https://github.com/concourse/rfcs/pull/77) and other options considered.

The current assumption would be that the registry is accessible by every K8s worker (including external workers).

### Executing Steps
Execute each step as its own standalone pod. In Concourse a step is the smallest executable abstraction. A pod is the smallest executable abstraction in K8s.

As a starting point, do something similar to the [k8s POC](https://github.com/concourse/concourse/issues/5209), use an `init` binary to keep the Pod from being deleted. The K8s worker client then monitors the state of the running Pods and executes actions on those Pods.

The K8s worker client will also use the K8s APIs to manage creation and cloning of volumes.

### Building and Using images
TODO

## Worker Lifecycle
The K8s runtime will continue using the Concourse API to register and heartbeat the Kubernetes worker. This provides the flexibility to extract the Kubernetes worker component in the future.

The **Worker Lifecycle component** would be responsible for the following;

  * **Registration**: The component reaches out to Kubernetes cluster. Registers with the ATC directly as a worker if it can successfully communicate with the Kubernetes API. This is a change as existing Garden/Containerd workers communicate with the ATC via the TSA. 
  * **Heartbeating/Running/Stalled**: The component will periodically ensure that the Kubernetes API is still reachable and heartbeat on behalf the K8s worker to the ATC. If it's no longer reachable then the heartbeat fails and the Kubernetes worker will be stalled by the ATC. 
  * **Land(ing/ed)**: Stop scheduling workloads on the worker.
  * **Retir(ing/ed)**: Stop scheduling workloads on the worker.
  * **Container GC**: The component would be responsible for cleaning up step pods that are no longer required by the web
  * **Volume GC**: The component would be responsible for cleaning up local **cache objects** that are no longer required by the web.
  * **Base Resources**: The Worker would advertise these base resources. This definition would include the list of base resources and their registry & repository metadata (eg. imagePullSecrets)

## Authenticating to the ATC API
The **Worker Lifecycle Component** should have its own identity (client id & secret) to communicate with the web API securely. 

Ideally, each instance of the component should have its own unique identity.

# Milestones
## Operator Use Cases
1. A K8s worker & external Concourse web (Simpler for local development)
   + register worker
   + heartbeat
1. Fly workers
1. Fly containers
1. Fly volumes
1. Pod GC'ing - only delete pods we know about. Ignore other pods.
1. Volume GC'ing
1. A K8s worker & in-cluster Concourse web 
1. Worker retiring/landing
   + fly land-worker
   + fly prune-worker
1. Tracing
1. Metrics (Placeholder)
1. External K8s worker that is not reachable by the web

## Developer Use Cases
1. Hello World (without `image_resource`)
   ```
   ---
   jobs:
   - name: job
     public: true
     plan:
     - task: simple-task
       config:
         platform: linux
         image_resource:
           type: registry-image
           source: { repository: busybox }
         run:
           path: echo
           args: ["Hello, world!"]
   ```
   * Task with params
   * Container limits
   * Privileged container
   * fly abort
   * stream/capture stdout & stderr
1. Hello World (2 tasks with inputs/outputs,without `image_resource`)
   ```
   ---
   jobs:
   - name: create-and-consume
     public: true
     plan:
     - task: make-a-file
       config:
       platform: linux
       image_resource:
         type: registry-image
         source: { repository: busybox }
       run:
         path: sh
         args:
         - -exc
         - ls -la; echo "Created a file on $(date)" > ./files/created_file
       outputs:
        - name: files
     - task: consume-the-file
       config:
         platform: linux
         image_resource:
           type: registry-image
           source: { repository: busybox }
         inputs:
         - name: files
         run:
           path: cat
           args:
           - ./files/created_file
   ```
   * Task with output
   * Task with input
   * Input mapping
   * Output mapping
   * Task cache
   * fly clear-task-cache
   * rootfs_uri ?
1. Fly Execute
   * `params`
   * worker `tag`
   * `--inputs-from`
   * upload inputs
   * image from a pipeline->job->step
   * outputs
1. Fly watch
1. Fly Hijack
1. Booklit Sample (Resources support)
   ```
   resources:
   - name: booklit
     type: git
     source: {uri: "https://github.com/vito/booklit"}

   jobs:
   - name: unit
     plan:
     - get: booklit
       trigger: true
   ```
   * Check step
      * logs (capture stderr)
      * abort
   * Get step
      * logs (capture stderr)
      * abort
1. Put step
   ```
   resources:
   - name: booklit
     type: git
     source: {uri: "https://github.com/vito/booklit"}

   - name: booklit-dev
     type: git
     source:
       uri: "https://github.com/vito/booklit"
       branch: dev

   jobs:
   - name: unit
     plan:
     - get: booklit
       trigger: true
     - put: booklit-dev
       params:
         repository: booklit
   ```
   * logs (capture stderr)
   * abort
1. Hello World (`Task.file`)
   ```
   resources:
   - name: my-repo
     type: git
     source: # ...

   jobs:
   - name: use-task-file
     plan:
     - get: my-repo
     - task: unit
       file: my-repo/ci/unit.yml
   ```
   * Task with vars
1. Hello World (Add support for `Task.image`)
   ```
   resources:
   - name: my-image
     type: registry-image
     source: {repository: golang, tag: "1.13"}
   jobs:
   - name: use-image
     plan:
     - get: my-image
     - task: unit
       image: my-image
   ```
1. Hello World (Add support for `image_resource`)
   ```
   ---
   jobs:
   - name: job
     public: true
     plan:
     - task: simple-task
       config:
         platform: linux
         image_resource:
           type: registry-image
           source: { repository: busybox }
         run:
           path: echo
           args: ["Hello, world!"]
   ```
1. 2 steps tagged with 2 workers (k8s to k8s)
1. 2 steps tagged with 2 workers on k8s and another runtime (K8s + containerd)
1. Hello World ( k8s Windows platform )
1. Resources ( k8s Windows platform )



# Open Questions

## Worker Lifecycle

* How should [worker `tags`](https://concourse-ci.org/concourse-worker.html#worker-configuration) be used? Should we pass the tag down to Kubernetes as the node name or not pass it down at all?
  * Option 1: We should not pass the tag down to Kubernetes. Tags are used in Concourse to select a set of workers and if we are treating Kubernetes as a worker then it should not operate on the tag(s) like workers currently behave.
  * Option 2: The purpose of tags is to control where steps are or aren't executed. K8s provides a few ways of achieving this, such as `nodeSelector` or more flexibile [Affinity & Anti-Affinity](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/)
* Should/could this component be run as a standalone component ?
	* The benefit of doing so would allow it to be managed separately from Concourse web. Scaling the web nodes is independent to scaling the worker lifecycle component. 
 
* Today the TSA component provides two services: 1) securing communication to and from the worker and 2) allowing a public web instance to talk to a woker inside a private network. With a Kubernetes worker communication is already secure. Is there some third-party tool we can leverage to achieve the second service that TSA currently provides us?


* Container Execution: Where do we store task step status similar to updating garden container properties to store exit status ?

## Authenticating to the k8s worker 
* How do we support different auth providers ?
* How do we support multiple worker configurations (across K8s clusters) using a Service Account ?

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


# Appendix

### Worker Mapping Alternatives
* Target an entire cluster. It would require Concourse targetting namespace to make a pod. We couldn't think of any compelling reasons to suggest this structure.
* Cluster + K8s Node. Concourse would bypass a lot of Kubernetes scheduling features with this option if Concourse decides which K8s node to place worklaods on.

### Boundary Where We Introduce Kubernetes Logic Alternatives
* Behind the Garden API, similar to `containerd`. [More details in our review of the k8s POC](https://github.com/concourse/concourse/issues/5986#issuecomment-675061559).


### Step to Pod Mapping Alternatives
* A job build could be mapped to a Pod. However, pods don't provide any mechanisms for ordering of containers which would be required for steps. This step planning logic is already provided by the `exec` engine and would remain in Concourse in order to support other runtimes.
* [SPIKE] Other K8s abstractions such as Jobs or other higher level abstractions such as Tekton TaskRuns could be explored

### Coordinating Container Execution Alternatives
* The ATC does not execute actions on the pod. Instead the Pod definition contains everything we want executed and Concourse observes the result.
  * Is live log streaming possible?
  * Is it possible to persist the Pod after the desired aciton completes so it can be intercepted later?
* [Porter](https://github.com/concourse/porter) Sidecar that uses external S3 compatible blobstore - sidecar container that monitors the execution of containers and coordinates them in the right order. Could be modified to use an image registry instead of S3 though.

