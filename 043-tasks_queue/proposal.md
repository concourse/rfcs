# Summary

We propose to enhance Concourse's task scheduling process. Instead of tasks immediately picking a compatible worker, a task is passed to a new component: the Resource Pool. 

The Resource Pool decides when the execution of a task is possible (contrary to the placement strategies which decide which worker is preferred). The Resource Pool component allows the implementation of specific, gradual improvements to making Concourse's scheduling safe, fair and scalable.


# Motivation

There have been multiple issues ([#2577](https://github.com/concourse/concourse/issues/2577), [#2928](https://github.com/concourse/concourse/issues/2928) etc) opened regarding Concourse's build/container scheduling and the container placement strategies currently present (`random`, `volume-locality`, `fewest-build-containers` and `limit-active-task`).

As Concourse operators, today we see that Concourse's scheduling process is:

1. not safe: workers are not protected from overload. When the system is overloaded it has no chance to recover.
2. not fair: tasks are racing for being executed without any specific order
3. not scalable: it is not possible to measure if the number of workers is appropriate to the current user demand.

To highlight the problem in more detail here's an overview of Concourse's task scheduling process.

1. the module scheduler selects the tasks that can be run (all their inputs are available, their job/pipeline is not paused)
2. the task submits itself to the WorkerPool
3. the WorkerPool selects a compatible worker according to the configured PlacementStrategy
4. the task runs

During this process no component asks whether the system, in particular the workers, are capable of accepting more tasks. The different placement strategy implementations only express preferences.

The scheduling process requires a new component which keeps track of the available resources as tasks start, execute and stop on the available workers.


# Proposal

Let's take a look at the RunTaskStep function from (`atc/worker/client.go`)

```go
func (client *client) RunTaskStep(...) TaskResult {
    //...
    chosenWorker, err := client.chooseTaskWorker(...)
    //...
}
```
This function is called from the `Run()` method of TaskStep, which is effectively responsible for starting the task's container.

The task chooses a worker given the placement strategy and the worker starts executing the corresponding container. This happens without taking into account the available resources on the worker. If users submit more jobs than the workers can instantaneously handle, the system becomes overloaded. Additionally, in case of overload no corrective action is possible since the tasks cannot re-schedule themselves.

Instead of directly allocating a worker from a task, we propose the following, extended worker allocation process:

![resource pool schema](https://ci-pix4d-internal-static.s3-eu-west-1.amazonaws.com/concourse-scheduling-rfc.png)

A new component, called Resource Pool is placed between the worker client (the ATC) and worker pool. The client would interact with the worker pool as follows:

1. The client requests a worker from Resource Pool.  This call blocks until a worker becomes free.
2. The resource pool returns the worker to the client.  The worker has been chosen from the available workers using the container placement strategy.
3. The client starts the task container on the allocated worker
4. The task container stops
5. The worker client releases the worker's resources

The key idea is the resource pool throttles the starting of containers and protects the workers from overload.

In this schema the resource pool only increases and decreases the number of active tasks on the workers.  In the next section we show that the proposal is not specific to the active tasks. The resource pool could be aware of other parameters that describe the workers' available capacity.

## Resource Pool

The new ResourcePool component has the following responsibilities:

* observe the number of active tasks on the worker
* offer workers' resources to tasks

Concretely, we propose the following interface
```go
type Resources = map[string]int  // for example: {"active-task": 1}

type ResourcePool interface {
    RequestWorker(requiredResources Resources, strategy PlacementStrategy, task parameters...) -> Worker
    ReleaseWorker(requiredResources Resources, worker Worker)
}
```

This interface mimics the fundamental operations of locks and semaphores.

Now, the interaction between the Client and the ResourcePool integrates unobtrusively into the current code base. Instead of tasks sending themselves immediately to the workers, the same part of the code could be, schematically

```go
func (client *client) RunTaskStep(...) TaskResult {
    //...

    requiredResources := make(map[string]int)
    requiredResources["active-task"] = 1
 
    chosenWorker := resourcePool.RequestWorker(requiredResoures, placementStrategy, taskParameters...)
    defer resourcePool.ReleaseWorker(requiredREsources, chosenWorker)

    //...now the worker can start the task container...
}
```

## Implementation as a task queue

This proposal doesn't want to implement a full-fledged workload scheduler into Concourse, but a simple mechanism by which the workers are protected from overload and each task gets a fair share of the workers' resources.

In the pull request [#4612](https://github.com/concourse/concourse/pull/4612) a task queue was proposed. This task was rejected because the implementation was too specific for the `limit-active-task` placement strategy. We believe, however, that the components from that PR are still valuable. At least, it's a good starting point.

The first resource pool implementation can be a simple task queue:

* The RequestWorker messages are queued up. The queue is stored in the database
* As resources become available they are offered to the first task in the queue.

This resource pool implementation would be independent of the chosen worker placement strategy. Thus all worker placement strategies benefit from a safe and fair task distribution process.

# Open Questions

* Is "Resource Pool" a good name? (resource means something different in the context of pipeline jobs and tasks)
* Could the functionality of the "Resource Pool"' be fit some already existing component?
* Should this be tasks scheduling or more generic step scheduling? That is to include check containers (resource-type checks etc.)
* What is the relationship to the new LIDAR component?
    * Resource check queue [#3788](https://github.com/concourse/concourse/issues/3788)
    * lidar: tracking checks in the database [#4202](https://github.com/concourse/concourse/issues/4202)


# Answered Questions

# New Implications

## Effect on PlacementStrategy

In the proposed schema the worker pool uses the provided worker placement strategy to choose among the workers where resources are available.  Contrast this to the current behavior where the placement strategy is applied on all the connected workers.

## Improved observability

This proposal improves the observability of the system's current load for both users and operators.

Users

* the system can display its current load by looking at the pending resource offers in the resource pool
* the system could potentially estimate the start time of the user's task.

Operators

* by counting the pending resource at any given time operators can evaluate if the size of the system corresponds to the current load
* if the number of pending resource requests is too high the more workers could be automatically added to the system.

## Fewer placement strategies
The limit-active-task placement strategy could be removed. Its implementation could be absorbed in the first resource pool implementation.

## On demand workers
It's possible to imagine a worker pool implementation which asks for the creation of a new worker in case there's none available in the system. We have to keep in mind that a Concourse system must be always able to run check containers.
