# Summary

An overview of how we plan to implement Kubernetes as a runtime for Concourse.

# Motivation

We want to leverage Kubernetes as a runtime for container orchestration. The K8s runtime would be specifically leveraged for;

- `stability` by offering more robust container placement 
- `cost savings` by offering simpler mechanisms for scaling the system up/down to match demand
- `security` by offering defined interfaces for auditing and policy management
- `observability` by leveraging logging & metrics solutions in the K8s ecosystem
 
# Proposal
## Storage
The k8s runtime will continue to use baggageclaim to provide volumes to containers. This will be provided by creating a Baggageclaim CSI Driver . [See RFC 74 for more details](https://github.com/concourse/rfcs/pull/77) and other options considered.

The current assumption would be that the registry is accessible by every K8s worker (including external workers).


## Worker Mapping
A K8s Concourse worker would be represented by a K8s worker + K8s namespace. This was the mapping suggested in the [k8s POC](https://github.com/concourse/concourse/issues/5209), where a [namespace](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/) in a [cluster](https://kubernetes.io/docs/concepts/architecture/) represented a single Concourse worker.

This leverages multi-tenant nature of Kubernetes and allows the Kubernetes cluster operator to manage and isolate Concourse workloads via the targeted namespace. It also allows an operator to configure capacity for a Concourse worker using [Resource Quotas](https://kubernetes.io/docs/tasks/administer-cluster/manage-resources/quota-memory-cpu-namespace/).

With this mapping a single Kubernetes cluster can represent multiple workers and manage resources based on namespaces.

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

## Authenticating to the k8s worker
Concourse would support both mechanisms for authenticating to the k8s cluster
- kubeconfig
- service account

## Authenticating to the web API
The **Worker Lifecycle Component** should have its own identity (client id & secret) to communicate with the web API securely. 

Ideally, each instance of the component should have its own unique identity. 

## Boundary Where We Introduce Kubernetes Logic
Same as the [k8s POC](https://github.com/concourse/concourse/issues/5209), implement the Kubernetes worker behind the [`worker.client`](https://github.com/concourse/concourse/blob/master/atc/worker/client.go).

## Step to Pod Mapping
Execute each step as its own standalone pod. In Concourse a step is the smallest executable abstraction. A pod is the smallest executable abstraction in K8s.

## Coordinating Container Execution
As a starting point, do something similar to the [k8s POC](https://github.com/concourse/concourse/issues/5209), use an `init` binary to keep the Pod from being deleted. ATC then monitors the state of the running Pods and executes actions on those Pods. The storage solution we end up going with will be a heavy driver of how we end up coordinating container execution with fetching and saving inputs/outputs.


# Milestones
## Operator Use Cases
1. 1 K8s worker & Concourse web external (Simpler for local development)
  + register worker
  + heartbeat
1. Fly workers
1. Fly containers
1. Fly volumes
1. Pod GC'ing - only delete pods we know about. Ignore other pods.
1. 1 K8s worker & Concourse web in-cluster
1. Image Registry GC'ing
1. Worker retiring/landing
  + fly land-worker
  + fly prune-worker
1. Tracing
1. Metrics (Placeholder)
1. Default K8s container placement strategy
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
  + Task with params
  + Container limits
  + Privileged container
  + fly abort
  + stream/capture stdout & stderr

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
  + Task with output
  + Task with input
  + Input mapping
  + Output mapping
  + Task cache
  + fly clear-task-cache
  + rootfs_uri ?

1. Fly Execute
  + `params`
  + worker `tag`
  + `--inputs-from`
  + upload inputs
  + image from a pipeline->job->step
  + outputs

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
  + Check step
    + logs (capture stderr)
    + abort
  + Get step
    + logs (capture stderr)
    + abort

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

  + logs (capture stderr)
  + abort

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
  + Task with vars

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

* How should [worker `tags`](https://concourse-ci.org/concourse-worker.html#worker-configuration) be used? Should we pass the tag down to Kubernetes as the node name or not pass it down at all?
  * Option 1: We should not pass the tag down to Kubernetes. Tags are used in Concourse to select a set of workers and if we are treating Kubernetes as a worker then it should not operate on the tag(s) like workers currently behave.
  * Option 2: The purpose of tags is to control where steps are or aren't executed. K8s provides a few ways of achieving this, such as `nodeSelector` or more flexibile [Affinity & Anti-Affinity](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/)
* Worker lifecycle: With volumes being stored in an image registry volumes are no longer associated with a specific worker. Should we change what it means to "Retire" a worker? This will be driven out by how we develop the storage solution.
* Worker lifecycle: Should/could this component be run as a standalone component ?
  * The benefit of doing so would allow it to be managed separately from Concourse web. Scaling the web nodes is independent to scaling the worker lifecycle component. 
* Today the TSA component provides two services: 1) securing communication to and from the worker and 2) allowing a public web instance to talk to a woker inside a private network. With a Kubernetes worker communication is already secure. Is there some third-party tool we can leverage to achieve the second service that TSA currently provides us?
* Container Execution: Where do we store task step status similar to updating garden container properties to store exit status ?
* Authenticating to the k8s worker
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

