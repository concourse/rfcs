# The Runtime Interface

## What is the Line Between Core and Runtime?

...

### Build Step Execution

The `exec` package is the cleanest place to discuss the process by which a scheduled build's plan is converted to executable steps, and handed off to the "runtime". The execution of steps involves the creation of containers and volumes by the "Runtime" (Currently most of this lives in the  `worker` package, but there's a higher-order abstraction for resources in the `resource` package, and the `db` package is also involved at different levels )

There are 3 builds step types which have a concrete execution involving Containers and Volumes, while the other step types available are meta-steps:

#### `get`

**Simplified:**

For a `get` step the Runtime requires the Resource Image (either a built-in resource RootFS living on the worker, or a cached image of a custom Resource Type) to create a container and run the `in` script. This will produce an `Artifact` (aka a Resource Cache Volume)

`( Resource Image ) -> ( Artifact )`

**Reality:**

[exec/get_step.go](https://github.com/concourse/concourse/blob/4cfe7e15f00de21d0aee7273867f75899cbeef20/atc/exec/get_step.go#L160-L196)

#### `task`

**Simplified:**

For a `task` step the Runtime requires Task Image (an `Artifact` produced by a previous step, or `image_resource`) to create a container and run the Task script, and any input `Artifact`s defined in the Task Config. Once the Task script successfully executes, there will be a set of output `Artifacts`s available to later steps.

`( Task Image + []Artifact ) -> ( []Artifact )`

**Reality:**

[exec/task_step.go](https://github.com/concourse/concourse/blob/master/atc/exec/task_step.go#L185-L295)

#### `put`

**Simplified:**

For a `put` step the Runtime requires the Resource Image (either a built-in resource RootFS living on the worker, or an `Artifact` of a custom Resource Type's image) to create a container and run the `out` script, and any `Inputs` defined on the `put` step. This will produce no `Artifact`s , as all that is required is the version produced by the step, which is sent over Stdout.

`( Resource Image + []Artifact ) -> ()`

**Reality:**

[exec/put_step.go](https://github.com/concourse/concourse/blob/master/atc/exec/put_step.go#L123-L166)


...


### Resource Checking

> But what about Resource checking! `radar` also creates Containers!

**Simplified:**

For resource checking, the Runtime portion only involves executing the `check` script in a Container based on the Resource Image (either a built-in resource RootFS living on the worker, or an `Artifact` of a custom Resource Type's image)

`( Resource Image ) -> ()`

**Reality:**

[radar/resource_scanner.go](https://github.com/concourse/concourse/blob/0cba188c2af6334842838ef64c38b81ea6a65eb8/atc/radar/resource_scanner.go#L310-L342)


## A Common Interface

In terms of what the Runtime produces, the most complex scenario is when an `Execution` (a vague idea of a container being created) consumes a set of `Artifact`s as Inputs, and produces a set of Artifacts as Outputs. Core has enough context to know how to differentiate these `Artifact`s before the `Execution`, and practically these are all bind-mounted Volumes in the case of the Garden Runtime.

For each step, Core also needs to be provided some handle on the `Result` of each step; output sent to `stdout`/`stderr`, an exit status, and some asyncronous way to know when to check the exit status to see whether the step should be considered failed or not.


```go
type Result struct {
   Stdout io.Reader
   Stderr io.Reader
   ExitStatus <-chan int
}
```

Currently we access information this in [many](https://github.com/concourse/concourse/blob/0cba188c2af6334842838ef64c38b81ea6a65eb8/atc/exec/task_step.go#L218-L221) [different](https://github.com/concourse/concourse/blob/master/atc/exec/put_step.go#L149-L152) [places](https://github.com/concourse/concourse/blob/master/atc/resource/run_script.go#L110-L133), and the reality is much more murky than this simple Common interface for `Results` of steps. 


We could build up a unified function which wraps up all the muddy abstractions leveraged by Core and build up cleaner boundaries between Core and "the executor of things given a set of artifacts".

```go
func Execute(run RunSpec, artifacts []Artifact) Result

type Artifact struct {
   ArtifactType string
   Path string
}

type RunSpec struct {
   Image Artifact
   Path string
   Args []string
   Env []string
}

```

* RunSpec is basically https://github.com/cloudfoundry/garden/blob/master/container.go#L136-L172 with a few less fields; we already know we want to BindMount the `Artifact`s, and `Image` will be an `Artifact` of type `"image"`

## Runtime's State

### How much state can / should the runtime store in the DB?

### Are the Current "Runtime" DB Objects Too Garden Specific?

## Efficiency Gains for Different Runtime Engines
